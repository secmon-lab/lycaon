import { ApolloClient, InMemoryCache, createHttpLink } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';

// HTTP connection to the API
const httpLink = createHttpLink({
  uri: '/graphql', // Always use relative path
  credentials: 'include', // Include cookies for authentication
});

// Middleware to add authentication headers
const authLink = setContext((_, { headers }) => {
  // Get the authentication token from local storage if it exists
  // For now, we'll rely on session cookies for authentication
  return {
    headers: {
      ...headers,
      // Add any additional headers here if needed
    },
  };
});

// Apollo Client instance
const client = new ApolloClient({
  link: authLink.concat(httpLink),
  cache: new InMemoryCache({
    typePolicies: {
      Query: {
        fields: {
          incidents: {
            // Define merge function for pagination
            keyArgs: ['first', 'after'],
            merge(existing, incoming, { args }) {
              // If this is a fresh query (no 'after' cursor), replace entirely
              if (!args?.after) {
                return incoming;
              }
              
              // If no existing data, return incoming
              if (!existing) return incoming;
              
              // For pagination, merge edges
              const merged = {
                ...incoming,
                edges: [...(existing.edges || []), ...(incoming.edges || [])],
              };
              
              return merged;
            },
          },
        },
      },
    },
  }),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'cache-and-network',
    },
  },
});

export default client;