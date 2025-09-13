import React, { useState } from 'react';
import { useQuery } from '@apollo/client/react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Chip,
  Typography,
  CircularProgress,
  Alert,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Visibility as ViewIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { format } from 'date-fns';
import { GET_INCIDENTS } from '../graphql/queries';

interface Incident {
  id: string;
  channelId: string;
  channelName: string;
  title: string;
  description: string;
  categoryId: string;
  status: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

interface IncidentEdge {
  node: Incident;
  cursor: string;
}

interface IncidentsData {
  incidents: {
    edges: IncidentEdge[];
    pageInfo: {
      hasNextPage: boolean;
      hasPreviousPage: boolean;
      startCursor: string | null;
      endCursor: string | null;
    };
    totalCount: number;
  };
}

const IncidentListPage: React.FC = () => {
  const navigate = useNavigate();
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(20);

  const { loading, error, data, refetch } = useQuery<IncidentsData>(
    GET_INCIDENTS,
    {
      variables: {
        first: rowsPerPage,
        after: null,
      },
      onError: (error) => {
        console.error('GraphQL Error:', error);
        console.error('Error details:', {
          message: error.message,
          networkError: error.networkError,
          graphQLErrors: error.graphQLErrors,
        });
      },
    }
  );

  const handleChangePage = (_event: unknown, newPage: number) => {
    setPage(newPage);
    // TODO: Implement cursor-based pagination
  };

  const handleChangeRowsPerPage = (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
    refetch({
      first: parseInt(event.target.value, 10),
      after: null,
    });
  };

  const handleViewIncident = (id: string) => {
    navigate(`/incidents/${id}`);
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'open':
        return 'error';
      case 'closed':
        return 'success';
      default:
        return 'default';
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error">
        Error loading incidents: {error.message}
      </Alert>
    );
  }

  const incidents = data?.incidents.edges.map((edge: IncidentEdge) => edge.node) || [];
  const totalCount = data?.incidents.totalCount || 0;

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          Incidents
        </Typography>
        <Tooltip title="Refresh">
          <IconButton onClick={() => refetch()}>
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>ID</TableCell>
              <TableCell>Title</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Category</TableCell>
              <TableCell>Channel</TableCell>
              <TableCell>Created At</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {incidents.map((incident: Incident) => (
              <TableRow
                key={incident.id}
                hover
                sx={{ cursor: 'pointer' }}
                onClick={() => handleViewIncident(incident.id)}
              >
                <TableCell>{incident.id}</TableCell>
                <TableCell>
                  <Typography variant="body2" fontWeight="medium">
                    {incident.title}
                  </Typography>
                  {incident.description && (
                    <Typography
                      variant="caption"
                      color="text.secondary"
                      sx={{
                        display: 'block',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                        maxWidth: '300px',
                      }}
                    >
                      {incident.description}
                    </Typography>
                  )}
                </TableCell>
                <TableCell>
                  <Chip
                    label={incident.status}
                    color={getStatusColor(incident.status)}
                    size="small"
                  />
                </TableCell>
                <TableCell>{incident.categoryId || '-'}</TableCell>
                <TableCell>{incident.channelName}</TableCell>
                <TableCell>
                  {format(new Date(incident.createdAt), 'yyyy-MM-dd HH:mm')}
                </TableCell>
                <TableCell>
                  <Tooltip title="View">
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleViewIncident(incident.id);
                      }}
                    >
                      <ViewIcon />
                    </IconButton>
                  </Tooltip>
                </TableCell>
              </TableRow>
            ))}
            {incidents.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  <Typography color="text.secondary" py={3}>
                    No incidents found
                  </Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
        <TablePagination
          rowsPerPageOptions={[10, 20, 50]}
          component="div"
          count={totalCount}
          rowsPerPage={rowsPerPage}
          page={page}
          onPageChange={handleChangePage}
          onRowsPerPageChange={handleChangeRowsPerPage}
        />
      </TableContainer>
    </Box>
  );
};

export default IncidentListPage;