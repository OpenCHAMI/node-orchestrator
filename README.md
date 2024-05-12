# Node Orchestrator (Experimental)

This experimental service and client are meant to explore standardized object structures that can be shareed independently of backend implementation.

In [nodes.go](/nodes.go) is the definition of a set of go structs with some struct tags that allow code in [main.go](/main.go) to automatically create a [jsonschema](https://json-schema.org/) representation of the object structure.

Once compiled, this go program will create the schemas in the schemas/ directory.  These could also be served through a special service endpoint so clients can refer to them directly.

[client.py](/clients/client.py) demonstrates how to somewhat automatically create a python client that can read the jsonschema files and automatically generate API calls for common CRUD operations.  Since the jsonschema file establishes a contract, the client can validate the object before trying to submit it to a remote endpoint.

## Usage

```bash
go build . && ./node-orchestrator schemas && ./node-orchestrator serve 
python3 -m venv .venv
 .venv/bin/pip install -r clients/requirements.txt
 .venv/bin/python clients/client.py --url http://localhost:8080 --schema-dir schemas create --file client/computenode.json  ComputeNode
 ```

Adjust [computenode.json](/clients/computenode.json) to explore creating and updating different kinds of nodes.
