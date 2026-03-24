package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// ServeCmd starts an MCP server on stdio for IDE integration (Cline, Roo Code).
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for IDE integration (Cline, Roo Code)",
	Long: `Starts a JSON-RPC 2.0 server over stdio following the Model Context Protocol (MCP).
This allows Cline or Roo Code to use zen-coder natively as a tool.`,
	RunE: startMCPServer,
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func startMCPServer(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			sendError(nil, -32700, "Parse error", nil)
			continue
		}

		handleRequest(req)
	}
}

func handleRequest(req JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		sendResponse(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]string{
				"name":    "zen-coder",
				"version": "1.0.0",
			},
		})

	case "tools/list":
		sendResponse(req.ID, map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "zen_run",
					"description": "Run the zen-coder pipeline to implement a feature or fix a bug.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"task": map[string]interface{}{
								"type":        "string",
								"description": "Description of the task to perform",
							},
							"project_path": map[string]interface{}{
								"type":        "string",
								"description": "Path to the project directory",
							},
						},
						"required": []string{"task"},
					},
				},
			},
		})

	case "tools/call":
		var params struct {
			Name      string `json:"name"`
			Arguments struct {
				Task        string `json:"task"`
				ProjectPath string `json:"project_path"`
			} `json:"arguments"`
		}
		json.Unmarshal(req.Params, &params)

		if params.Name == "zen_run" {
			// Set flags and call the runPipeline internal function
			// Note: In an MCP server, we should ideally not use global flags, 
			// but for this implementation we simulate a CLI call.
			flagProject = params.Arguments.ProjectPath
			flagOutputMode = "json"
			flagNoEdit = true // IDEs don't want interactive editors usually
			
			// Capture stdout to return as tool output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the pipeline
			err := runPipeline(nil, []string{params.Arguments.Task})
			
			w.Close()
			os.Stdout = oldStdout
			
			var out []byte
			out, _ = io.ReadAll(r)

			if err != nil {
				sendResponse(req.ID, map[string]interface{}{
					"content": []map[string]string{
						{"type": "text", "text": fmt.Sprintf("Error: %v\nOutput: %s", err, string(out))},
					},
					"isError": true,
				})
			} else {
				sendResponse(req.ID, map[string]interface{}{
					"content": []map[string]string{
						{"type": "text", "text": string(out)},
					},
				})
			}
		} else {
			sendError(req.ID, -32601, "Method not found", nil)
		}

	case "notifications/initialized":
		// No-op

	default:
		sendError(req.ID, -32601, "Method not found", nil)
	}
}

func sendResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func sendError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: map[string]interface{}{
			"code":    code,
			"message": message,
			"data":    data,
		},
	}
	dataBytes, _ := json.Marshal(resp)
	fmt.Println(string(dataBytes))
}
