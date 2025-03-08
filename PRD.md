# Initial prompt
I'm going to create an MCP server that can let LLM to interact with go debugger, you can search for relevant information via https://github.com/go-delve/delve/tree/master/Documentation/api/dap and https://github.com/modelcontextprotocol/servers. 

I need you to use @https://github.com/mark3labs/mcp-go  for MCP implementation, which is go native.

the mcp-go's type support is incomplete, such that it does not have WithArray, WithObject. I've made a local copy at mcp-go/mcp/tools.go. Please add WithObject according to MCP specification here @https://github.com/modelcontextprotocol/specification/blob/main/schema/2024-11-05/schema.json 