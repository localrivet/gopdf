// gopdf-mcp-server-go/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	// Correct import for the library we built
	"slices"

	mcp "github.com/localrivet/gomcp"
)

var runnerPath string // Global variable to store runner path

// Define the structure for the arguments expected by our tool
type GeneratePdfArgs struct {
	Input        string   `json:"input"`
	Output       string   `json:"output"`
	InputType    string   `json:"inputType,omitempty"`
	Theme        string   `json:"theme,omitempty"`
	Footer       string   `json:"footer,omitempty"`
	Header       string   `json:"header,omitempty"`
	Cover        string   `json:"cover,omitempty"`
	SkipH1H2     bool     `json:"skipH1H2,omitempty"`
	MarginTop    string   `json:"marginTop,omitempty"`
	MarginBottom string   `json:"marginBottom,omitempty"`
	MarginLeft   string   `json:"marginLeft,omitempty"`
	MarginRight  string   `json:"marginRight,omitempty"`
	PageSize     string   `json:"pageSize,omitempty"`
	Orientation  string   `json:"orientation,omitempty"`
	Title        string   `json:"title,omitempty"`
	Replace      []string `json:"replace,omitempty"`
}

// Define the generate_pdf tool using mcp.ToolDefinition
var generatePdfTool = mcp.ToolDefinition{
	Name:        "generate_pdf",
	Description: "Generates a PDF from a Markdown or HTML file using gopdf-runner.",
	InputSchema: mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]mcp.PropertyDetail{
			"input":        {Type: "string", Description: "Raw Markdown or HTML content string"}, // Updated description
			"output":       {Type: "string", Description: "Path for output PDF file"},
			"inputType":    {Type: "string", Description: "Input type ('markdown' or 'html')"},
			"theme":        {Type: "string", Description: "Path to CSS theme file (optional)"},
			"footer":       {Type: "string", Description: "Path to footer HTML file (optional)"},
			"header":       {Type: "string", Description: "Path to header HTML file (optional)"},
			"cover":        {Type: "string", Description: "Path to cover HTML file (optional)"},
			"skipH1H2":     {Type: "boolean", Description: "Skip first H1/H2 in Markdown"},
			"marginTop":    {Type: "string", Description: "Top margin (e.g., '25mm')"},
			"marginBottom": {Type: "string", Description: "Bottom margin"},
			"marginLeft":   {Type: "string", Description: "Left margin"},
			"marginRight":  {Type: "string", Description: "Right margin"},
			"pageSize":     {Type: "string", Description: "Page size (e.g., 'Letter', 'A4')"},
			"orientation":  {Type: "string", Description: "Orientation ('Portrait', 'Landscape')"},
			"title":        {Type: "string", Description: "Document title metadata"},
			"replace":      {Type: "array", Description: "Replacements (key=value pairs)"}, // Simplified schema for example
		},
		Required: []string{"input", "output"},
	},
	OutputSchema: mcp.ToolOutputSchema{
		Type:        "object", // Return status and output path/error
		Description: "Result of the PDF generation containing status and output path or error message.",
	},
}

// Tool registry for this server
var toolRegistry = map[string]mcp.ToolDefinition{
	generatePdfTool.Name: generatePdfTool,
}

// handleToolDefinitionRequest sends the list of defined tools.
func handleToolDefinitionRequest(conn *mcp.Connection) error {
	log.Println("Handling ToolDefinitionRequest")
	tools := make([]mcp.ToolDefinition, 0, len(toolRegistry))
	for _, tool := range toolRegistry {
		tools = append(tools, tool)
	}
	responsePayload := mcp.ToolDefinitionResponsePayload{Tools: tools}
	return conn.SendMessage(mcp.MessageTypeToolDefinitionResponse, responsePayload)
}

// handleUseToolRequest handles the execution of the generate_pdf tool.
func handleUseToolRequest(conn *mcp.Connection, requestPayload *mcp.UseToolRequestPayload) error {
	log.Printf("Handling UseToolRequest for tool: %s", requestPayload.ToolName)

	if requestPayload.ToolName != generatePdfTool.Name {
		log.Printf("Tool not found: %s", requestPayload.ToolName)
		return conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{
			Code:    "ToolNotFound",
			Message: fmt.Sprintf("Tool '%s' not found", requestPayload.ToolName),
		})
	}

	// --- Execute generate_pdf ---
	var args GeneratePdfArgs
	// Need to marshal the interface{} map back to JSON and then unmarshal to struct
	// Or iterate and type assert carefully. Let's try marshal/unmarshal.
	argsBytes, err := json.Marshal(requestPayload.Arguments)
	if err != nil {
		log.Printf("Error marshalling arguments: %v", err)
		return conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "InvalidPayload", Message: "Cannot process arguments map"})
	}
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		log.Printf("Error unmarshalling arguments into GeneratePdfArgs: %v", err)
		return conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "InvalidArgument", Message: fmt.Sprintf("Invalid arguments structure: %v", err)})
	}

	// Validate required arguments
	if args.Input == "" || args.Output == "" {
		return conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "InvalidArgument", Message: "Missing required arguments: input and output paths are required."})
	}

	// Construct command-line arguments
	cmdArgs := []string{
		fmt.Sprintf("-input=%s", args.Input),
		fmt.Sprintf("-output=%s", args.Output),
	}
	// ... (append other optional arguments as before) ...
	if args.InputType != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-inputType=%s", args.InputType))
	}
	if args.Theme != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-theme=%s", args.Theme))
	}
	if args.Footer != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-footer=%s", args.Footer))
	}
	if args.Header != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-header=%s", args.Header))
	}
	if args.Cover != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-cover=%s", args.Cover))
	}
	if args.SkipH1H2 {
		cmdArgs = append(cmdArgs, "-skipH1H2")
	}
	if args.MarginTop != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-marginTop=%s", args.MarginTop))
	}
	if args.MarginBottom != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-marginBottom=%s", args.MarginBottom))
	}
	if args.MarginLeft != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-marginLeft=%s", args.MarginLeft))
	}
	if args.MarginRight != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-marginRight=%s", args.MarginRight))
	}
	if args.PageSize != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-pageSize=%s", args.PageSize))
	}
	if args.Orientation != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-orientation=%s", args.Orientation))
	}
	if args.Title != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-title=%s", args.Title))
	}
	for _, rep := range args.Replace {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-replace=%s", rep))
	}

	// Execute the runner
	log.Printf("Executing runner: %s %v", runnerPath, cmdArgs)
	cmd := exec.Command(runnerPath, cmdArgs...)
	cmd.Stderr = os.Stderr
	outputBytes, err := cmd.Output() // Captures stdout

	if err != nil {
		errMsg := fmt.Sprintf("Error executing gopdf-runner: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			errMsg = fmt.Sprintf("Error executing gopdf-runner: %v. Stderr: %s", err, string(exitErr.Stderr))
		}
		log.Printf(errMsg)
		// Send error via MCP Error message
		return conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{
			Code:    "ToolExecutionError",
			Message: errMsg,
		})
	}

	// Success
	outputFilePath := strings.TrimSpace(string(outputBytes))
	log.Printf("Successfully generated PDF: %s", outputFilePath)
	responsePayload := mcp.UseToolResponsePayload{
		Result: map[string]interface{}{ // Return a structured result
			"status":     "success",
			"outputFile": outputFilePath,
		},
	}
	return conn.SendMessage(mcp.MessageTypeUseToolResponse, responsePayload)
}

func main() {
	// Determine runner path
	serverExecutablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting server executable path: %v", err)
	}
	serverDir := filepath.Dir(serverExecutablePath)
	// Adjust relative path based on your actual project structure
	runnerPath = filepath.Join(serverDir, "..", "bin", "gopdf-runner") // Example path
	// Check if runner exists
	if _, err := os.Stat(runnerPath); os.IsNotExist(err) {
		log.Fatalf("gopdf-runner not found at expected path: %s", runnerPath)
	} else if err != nil {
		log.Fatalf("Error checking runner path: %v", err)
	}

	// Log to stderr
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Println("Starting GoPdf MCP Server...")

	serverName := "gopdf-mcp-server-go"
	conn := mcp.NewStdioConnection()

	// --- Perform Handshake ---
	log.Println("Waiting for HandshakeRequest...")
	msg, err := conn.ReceiveMessage()
	if err != nil {
		log.Fatalf("Failed to receive initial message: %v", err)
	}
	if msg.MessageType != mcp.MessageTypeHandshakeRequest {
		errMsg := fmt.Sprintf("Expected HandshakeRequest, got %s", msg.MessageType)
		_ = conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "HandshakeFailed", Message: errMsg})
		log.Fatal(errMsg)
	}
	var hsReqPayload mcp.HandshakeRequestPayload
	err = mcp.UnmarshalPayload(msg.Payload, &hsReqPayload)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to unmarshal HandshakeRequest payload: %v", err)
		_ = conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "HandshakeFailed", Message: errMsg})
		log.Fatalf(errMsg)
	}
	log.Printf("Received HandshakeRequest from client: %s", hsReqPayload.ClientName)
	// Basic version check (assuming client sends "1.0")
	clientSupportsCurrent := slices.Contains(hsReqPayload.SupportedProtocolVersions, mcp.CurrentProtocolVersion)
	if !clientSupportsCurrent {
		errMsg := fmt.Sprintf("Client does not support protocol version %s", mcp.CurrentProtocolVersion)
		_ = conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "UnsupportedProtocolVersion", Message: fmt.Sprintf("Server requires protocol version %s", mcp.CurrentProtocolVersion)})
		log.Fatal(errMsg)
	}
	// Send HandshakeResponse
	hsRespPayload := mcp.HandshakeResponsePayload{SelectedProtocolVersion: mcp.CurrentProtocolVersion, ServerName: serverName}
	err = conn.SendMessage(mcp.MessageTypeHandshakeResponse, hsRespPayload)
	if err != nil {
		log.Fatalf("Failed to send HandshakeResponse: %v", err)
	}
	log.Printf("Handshake successful with client: %s", hsReqPayload.ClientName)
	// --- End Handshake ---

	// --- Main Message Loop ---
	log.Println("Entering main message loop...")
	for {
		msg, err := conn.ReceiveMessage()
		if err != nil {
			if err.Error() == "failed to read message line: EOF" || strings.Contains(err.Error(), "EOF") {
				log.Println("Client disconnected (EOF received). Server shutting down.")
			} else {
				log.Printf("Error receiving message: %v. Server shutting down.", err)
			}
			break
		}

		log.Printf("Received message type: %s", msg.MessageType)
		var handlerErr error

		switch msg.MessageType {
		case mcp.MessageTypeToolDefinitionRequest:
			handlerErr = handleToolDefinitionRequest(conn) // Pass only conn
		case mcp.MessageTypeUseToolRequest:
			var utReqPayload mcp.UseToolRequestPayload
			err := mcp.UnmarshalPayload(msg.Payload, &utReqPayload)
			if err != nil {
				log.Printf("Error unmarshalling UseToolRequest payload: %v", err)
				handlerErr = conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "InvalidPayload", Message: fmt.Sprintf("Failed to unmarshal UseToolRequest payload: %v", err)})
			} else {
				handlerErr = handleUseToolRequest(conn, &utReqPayload) // Pass parsed payload
			}
		default:
			log.Printf("Handler not implemented for message type: %s", msg.MessageType)
			handlerErr = conn.SendMessage(mcp.MessageTypeError, mcp.ErrorPayload{Code: "NotImplemented", Message: fmt.Sprintf("Message type '%s' not implemented by server", msg.MessageType)})
		}

		if handlerErr != nil {
			log.Printf("Error handling message type %s: %v", msg.MessageType, handlerErr)
			if strings.Contains(handlerErr.Error(), "write") || strings.Contains(handlerErr.Error(), "pipe") {
				log.Println("Detected write error, assuming client disconnected. Shutting down.")
				break
			}
		}
	}
	log.Println("Server finished.")
}
