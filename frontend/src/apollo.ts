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
            keyArgs: false,
            merge(existing, incoming) {
              if (!existing) return incoming;
              
              // Handle pagination merge
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