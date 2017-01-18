# batched-graphql-handler
An [http handler](https://golang.org/pkg/net/http/#Handler) to use [graphql-go](https://github.com/neelance/graphql-go) with a graphql client that supports batching like [graphql-query-batcher](https://github.com/nicksrandall/graphql-query-batcher) or  [apollo client](https://github.com/apollostack/apollo-client).

### Notes
- This handler only supports [batched queries](https://dev-blog.apollodata.com/query-batching-in-apollo-63acfd859862#.p7459gedh). It doesn't support all of the apollo stack featuers. 
- If you'd like to support gzip in your handler, I suggest wrapping this handler with [GZIP Handler](https://github.com/NYTimes/gziphandler) by NY Times
