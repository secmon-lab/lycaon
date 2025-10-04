package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack"
)

// getTestCategoriesForIncident returns categories for incident testing purposes

// Helper function to create a test model.Config
func testConfig() *model.Config {
	return &model.Config{
		Categories: []model.Category{
			{
				ID:           "security_incident",
				Name:         "Security Incident",
				Description:  "Security-related incidents",
				InviteUsers:  []string{"@security-lead"},
				InviteGroups: []string{"@security-team"},
			},
			{
				ID:          "system_failure",
				Name:        "System Failure",
				Description: "System or service failures and outages",
			},
			{
				ID:          "unknown",
				Name:        "Unknown",
				Description: "Unknown incidents",
			},
		},
	}
}

func TestIncidentUseCaseCreateIncident(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful incident creation", func(t *testing.T) {
		// Use memory repository for testing
		repo := repository.NewMemory()

		// Create mock Slack client with default behavior
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T123456",
					Team:   "Test Team",
					UserID: "U123456",
					User:   "test-bot",
				}, nil
			},
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}

		// Create use case with mock and default categories
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Create an incident
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "database outage",
			Description:       "",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()
		gt.Equal(t, 1, incident.ID)
		gt.Equal(t, "inc-1-database-outage", incident.ChannelName)
		gt.Equal(t, "database outage", incident.Title)
		gt.Equal(t, "C-ORIGIN", incident.OriginChannelID)
		gt.Equal(t, "general", incident.OriginChannelName)
		gt.Equal(t, "U-CREATOR", incident.CreatedBy)

		// Verify channel ID was set by mock
		gt.Equal(t, "C-NEW-INCIDENT", incident.ChannelID)

		// Verify incident was saved to repository
		savedIncident, err := repo.GetIncident(ctx, 1)
		gt.NoError(t, err).Required()
		gt.Equal(t, incident.ID, savedIncident.ID)
	})

	t.Run("Multiple incidents get sequential IDs", func(t *testing.T) {
		repo := repository.NewMemory()
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T123456",
					Team:   "Test Team",
					UserID: "U123456",
					User:   "test-bot",
				}, nil
			},
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Create first incident
		incident1, _ := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "api error",
			Description:       "",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C1",
			OriginChannelName: "channel1",
			CreatedBy:         "U1",
		})
		gt.Equal(t, 1, incident1.ID)
		gt.Equal(t, "inc-1-api-error", incident1.ChannelName)

		// Create second incident
		incident2, _ := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "database down",
			Description:       "",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C2",
			OriginChannelName: "channel2",
			CreatedBy:         "U2",
		})
		gt.Equal(t, 2, incident2.ID)
		gt.Equal(t, "inc-2-database-down", incident2.ChannelName)

		// Create third incident
		incident3, _ := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "",
			Description:       "",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C3",
			OriginChannelName: "channel3",
			CreatedBy:         "U3",
		})
		gt.Equal(t, 3, incident3.ID)
		gt.Equal(t, "inc-3", incident3.ChannelName)
	})

	t.Run("GetIncident retrieves correct incident", func(t *testing.T) {
		repo := repository.NewMemory()
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T123456",
					Team:   "Test Team",
					UserID: "U123456",
					User:   "test-bot",
				}, nil
			},
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Create an incident
		created, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "test incident",
			Description:       "",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C-TEST",
			OriginChannelName: "test-channel",
			CreatedBy:         "U-TEST",
		})
		gt.NoError(t, err).Required()

		// Retrieve the incident
		retrieved, err := uc.GetIncident(ctx, created.ID.Int())
		gt.NoError(t, err).Required()
		gt.Equal(t, created.ID, retrieved.ID)
		gt.Equal(t, created.ChannelName, retrieved.ChannelName)
		gt.Equal(t, created.OriginChannelID, retrieved.OriginChannelID)
		gt.Equal(t, created.CreatedBy, retrieved.CreatedBy)
	})

	t.Run("GetIncident returns error for non-existent ID", func(t *testing.T) {
		repo := repository.NewMemory()
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T123456",
					Team:   "Test Team",
					UserID: "U123456",
					User:   "test-bot",
				}, nil
			},
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Try to get non-existent incident
		incident, err := uc.GetIncident(ctx, 999)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("incident not found")
	})
}

func TestIncidentUseCaseWithMockRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("Repository error handling", func(t *testing.T) {
		// Create mock repository
		mockRepo := &mocks.RepositoryMock{
			GetNextIncidentNumberFunc: func(ctx context.Context) (types.IncidentID, error) {
				return 0, goerr.New("database error")
			},
		}

		mockSlack := &mocks.SlackClientMock{}
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
		uc := usecase.NewIncident(mockRepo, mockSlack, testConfig(), nil, config)

		// Try to create incident - should fail due to repository error
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "test",
			Description:       "",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C1",
			OriginChannelName: "channel1",
			CreatedBy:         "U1",
		})
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("failed to get next incident number")
	})

	t.Run("Create incident with category invitations", func(t *testing.T) {
		// Use memory repository for testing
		repo := repository.NewMemory()

		// Track invited users
		var invitedUsers []string

		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T123456",
					Team:   "Test Team",
					UserID: "U123456",
					User:   "test-bot",
				}, nil
			},
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-INCIDENT-WITH-INVITES",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Capture invited users
				invitedUsers = append(invitedUsers, users...)
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}

		// Create mock Invite that tracks invitations
		mockInvite := &mocks.InviteMock{
			InviteUsersByListFunc: func(ctx context.Context, users []string, groups []string, channelID types.ChannelID) (*model.InvitationResult, error) {
				// Verify correct category invitations are passed
				gt.Equal(t, 1, len(users))
				gt.Equal(t, "@security-lead", users[0])
				gt.Equal(t, 1, len(groups))
				gt.Equal(t, "@security-team", groups[0])

				return &model.InvitationResult{
					Details: []model.InviteDetail{
						{
							UserID:       "U-SECURITY-LEAD",
							Username:     "@security-lead",
							SourceConfig: "@security-lead",
							Status:       "success",
						},
						{
							UserID:       "U-SECURITY-MEMBER1",
							Username:     "",
							SourceConfig: "@security-team",
							Status:       "success",
						},
					},
				}, nil
			},
		}

		// Create use case with mock invite
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), mockInvite, config)

		// Create an incident with security_incident category
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "security breach",
			Description:       "Potential security incident detected",
			CategoryID:        "security_incident",
			SeverityID:        "",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()
		gt.Equal(t, "security_incident", incident.CategoryID)

		// Verify invite was called with correct parameters
		gt.Equal(t, 1, len(mockInvite.InviteUsersByListCalls()))
		inviteCall := mockInvite.InviteUsersByListCalls()[0]
		gt.Equal(t, []string{"@security-lead"}, inviteCall.Users)
		gt.Equal(t, []string{"@security-team"}, inviteCall.Groups)
		gt.Equal(t, incident.ChannelID, inviteCall.ChannelID)
	})

	t.Run("Create incident with no category invitations", func(t *testing.T) {
		// Use memory repository for testing
		repo := repository.NewMemory()

		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T123456",
					Team:   "Test Team",
					UserID: "U123456",
					User:   "test-bot",
				}, nil
			},
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-INCIDENT-NO-INVITES",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}

		// Create mock Invite that should NOT be called
		mockInvite := &mocks.InviteMock{
			InviteUsersByListFunc: func(ctx context.Context, users []string, groups []string, channelID types.ChannelID) (*model.InvitationResult, error) {
				t.Fatal("InviteUsersByList should not be called for unknown category")
				return nil, nil
			},
		}

		// Create use case with mock invite
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), mockInvite, config)

		// Create an incident with unknown category (no invitations)
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "unknown issue",
			Description:       "Some unknown issue",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()
		gt.Equal(t, "unknown", incident.CategoryID)

		// Verify invite was NOT called
		gt.Equal(t, 0, len(mockInvite.InviteUsersByListCalls()))
	})
}

func TestIncidentUseCaseWithCustomPrefix(t *testing.T) {
	ctx := context.Background()

	t.Run("Create incident with custom prefix generates correct channel name", func(t *testing.T) {
		// Use memory repository for testing
		repo := repository.NewMemory()

		// Create mock Slack client
		var createdChannelName string
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T123456",
					Team:   "Test Team",
				}, nil
			},
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				// Capture the channel name that was requested
				createdChannelName = params.ChannelName
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}

		// Test with custom prefix "security"
		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("security"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Create an incident
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "data breach",
			Description:       "Suspicious data access detected",
			CategoryID:        "security_incident",
			SeverityID:        "",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "security-alerts",
			CreatedBy:         "U-SECURITY-ANALYST",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()

		// Verify the channel name uses the custom prefix
		gt.Equal(t, "security-1-data-breach", incident.ChannelName.String())
		gt.Equal(t, "security-1-data-breach", createdChannelName)

		// Verify other properties
		gt.Equal(t, "data breach", incident.Title)
		gt.Equal(t, "security_incident", incident.CategoryID)
		gt.Equal(t, "C-NEW-INCIDENT", incident.ChannelID.String())
	})

	t.Run("Create incident with different custom prefixes", func(t *testing.T) {
		testCases := []struct {
			name           string
			prefix         string
			title          string
			expectedPrefix string
		}{
			{"Alert prefix", "alert", "system down", "alert-1-system-down"},
			{"Incident prefix", "incident", "api failure", "incident-1-api-failure"},
			{"Emergency prefix", "emergency", "critical issue", "emergency-1-critical-issue"},
			{"Empty prefix fallback", "", "test issue", "inc-1-test-issue"}, // Should fallback to default
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Use fresh repository for each test to ensure incident ID starts at 1
				repo := repository.NewMemory()

				var createdChannelName string
				mockSlack := &mocks.SlackClientMock{
					AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
						return &slack.AuthTestResponse{TeamID: "T123456"}, nil
					},
					CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
						createdChannelName = params.ChannelName
						return &slack.Channel{
							GroupConversation: slack.GroupConversation{
								Conversation: slack.Conversation{ID: "C-NEW"},
								Name:         params.ChannelName,
							},
						}, nil
					},
					SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
						return &slack.Channel{}, nil
					},
					InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
						return &slack.Channel{}, nil
					},
					PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
						return "channel", "timestamp", nil
					},
				}

				var config *usecase.IncidentConfig
				if tc.prefix == "" {
					// Test default value by not specifying prefix
					config = usecase.NewIncidentConfig()
				} else {
					config = usecase.NewIncidentConfig(usecase.WithChannelPrefix(tc.prefix))
				}
				uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

				// Create an incident
				incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
					Title:             tc.title,
					Description:       "Test description",
					CategoryID:        "unknown",
					SeverityID:        "",
					OriginChannelID:   "C-ORIGIN",
					OriginChannelName: "general",
					CreatedBy:         "U-CREATOR",
				})

				// Verify incident was created successfully
				gt.NoError(t, err).Required()
				gt.V(t, incident).NotNil()

				// Verify the channel name uses the expected prefix
				gt.Equal(t, tc.expectedPrefix, incident.ChannelName.String())
				gt.Equal(t, tc.expectedPrefix, createdChannelName)
			})
		}
	})
}

func TestIncidentUseCaseWithBookmark(t *testing.T) {
	ctx := context.Background()

	t.Run("Add bookmark when frontend URL is configured", func(t *testing.T) {
		repo := repository.NewMemory()

		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-TEST-CHANNEL",
						},
					},
				}, nil
			},
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T-TEST-TEAM",
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			AddBookmarkFunc: func(ctx context.Context, channelID, title, link string) error {
				return nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return channelID, "1234567890.123456", nil
			},
		}

		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"), usecase.WithFrontendURL("https://lycaon.example.com"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Create an incident
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "Test Incident",
			Description:       "Test description",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()

		// Verify AddBookmark was called once with correct arguments
		addBookmarkCalls := mockSlack.AddBookmarkCalls()
		gt.Equal(t, 1, len(addBookmarkCalls))
		bookmarkCall := addBookmarkCalls[0]
		gt.Equal(t, "C-TEST-CHANNEL", bookmarkCall.ChannelID)
		gt.Equal(t, "Incident #1 - Web UI", bookmarkCall.Title)
		gt.Equal(t, "https://lycaon.example.com/incidents/1", bookmarkCall.Link)
	})

	t.Run("Skip bookmark when frontend URL is not configured", func(t *testing.T) {
		repo := repository.NewMemory()

		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-TEST-CHANNEL",
						},
					},
				}, nil
			},
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T-TEST-TEAM",
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			AddBookmarkFunc: func(ctx context.Context, channelID, title, link string) error {
				t.Error("AddBookmark should not be called when frontend URL is not configured")
				return nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return channelID, "1234567890.123456", nil
			},
		}

		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc")) // No frontend URL
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Create an incident
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "Test Incident",
			Description:       "Test description",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()

		// Verify AddBookmark was not called
		gt.Equal(t, 0, len(mockSlack.AddBookmarkCalls()))
	})

	t.Run("Handle bookmark failure gracefully", func(t *testing.T) {
		repo := repository.NewMemory()

		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-TEST-CHANNEL",
						},
					},
				}, nil
			},
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					TeamID: "T-TEST-TEAM",
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			AddBookmarkFunc: func(ctx context.Context, channelID, title, link string) error {
				return goerr.New("bookmark API failed")
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return channelID, "1234567890.123456", nil
			},
		}

		config := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"), usecase.WithFrontendURL("https://lycaon.example.com"))
		uc := usecase.NewIncident(repo, mockSlack, testConfig(), nil, config)

		// Create an incident - should succeed even if bookmark fails
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "Test Incident",
			Description:       "Test description",
			CategoryID:        "unknown",
			SeverityID:        "",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully despite bookmark failure
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()

		// Verify AddBookmark was called (and failed)
		gt.Equal(t, 1, len(mockSlack.AddBookmarkCalls()))
	})
}
