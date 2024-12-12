# Alfred Airtable Workflow

🚧 **Work in progress** 🚧

An Alfred workflow for managing Airtable records, adapted to **my personal Airtable database for link collection**.

The database consists of two tables: **Links** and **Lists**.
Where **Links** is the main table, and **Lists** is used to group links under specific topics.

The workflow supports adding, editing, searching, and filtering records.

## To-Do

- Rewrite the workflow in Go
    - [ ] Authentication and oauth flow
    - [ ] Data fetching with Airtale API
        - Concurrent requests to speed up fetching
        - Sync in the background without blocking the UI
        - Use the field `Last Modified` to fetch only the updated records
    - [ ] Data caching with sqlite
        - Keep the auth token and last fetched time in the cache
    - [ ] Adding, editing, deleting records with Airtable API
    - [ ] User interface: adding, editing, searching, filtering

## References

- [Airtable API](https://airtable.com/developers/web/api/introduction)
