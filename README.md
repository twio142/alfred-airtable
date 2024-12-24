# Alfred Airtable Workflow

🚧 **Work in progress** 🚧

An Alfred workflow for managing Airtable records, adapted to **my personal Airtable database for link collection**.

The database consists of two tables: **Links** and **Lists**.
Where **Links** is the main table, and **Lists** is used to group links under specific topics.

The workflow supports adding, editing, searching, and filtering records.

## To-Do

- Testing in Alfred
    - Why does `fetchRecords` sometimes fail to paginate?
- Rewrite the workflow in Go
    - [x] Authentication and OAuth flow (tested)
        1. Start the HTTP server in a goroutine: This server will handle the OAuth flow and listen for the callback from the OAuth provider.
        2. Use channels for communication: Create channels to pass the access token and any errors between the goroutines.
        3. Request the OAuth authorization: In the main goroutine, initiate the OAuth authorization request and wait for the response via the channel.
        4. Handle the OAuth callback: When the OAuth provider redirects back to your server, handle the callback, exchange the authorization code for an access token, and send the token back through the channel.
    - [x] Data fetching with Airtale API (tested)
        - Concurrent requests to speed up fetching
        - Sync in the background without blocking the UI
        - Use the field `Last Modified` to fetch only the updated records
    - [x] Data caching with sqlite (tested)
        - Keep the access token and last fetched time in the cache
    - [x] Adding, editing, deleting records with Airtable API (tested)
    - [x] User interface: adding, editing, searching, filtering (partially tested)
        - Think over the interaction logic in `editLink`

## References

- [Airtable API](https://airtable.com/developers/web/api/introduction)
