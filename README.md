# Node Orchestrator (Experimental)

This service and client showcase a set of experiments that may be useful for OpenCHAMI

## Jsonschema for object definitions

The jsonschema library we use supports reflecting go structs as jsonschema objects using struct tags.  Notice the tags in the struct below `jsonschema:"required"` indicates that the jsonschema of the BMC should require both Username and Password in order to be valid.

```go
type BMC struct {
	ID          uuid.UUID       `json:"id,omitempty" format:"uuid"`
	XName       xnames.BMCXname `json:"xname,omitempty"`
	Username    string          `json:"username" jsonschema:"required"`
	Password    string          `json:"password" jsonschema:"required"`
	IPv4Address string          `json:"ipv4_address,omitempty" format:"ipv4"`
	IPv6Address string          `json:"ipv6_address,omitempty" format:"ipv6"`
	MACAddress  string          `json:"mac_address" format:"mac-address" binding:"required"`
	Description string          `json:"description,omitempty"`
}
```

We can extend that behavior with some custom methods on the structs.  For example, we can extend the jsonschema definition of an xname with a regex for validation.

```go
func (NodeXname) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "NodeXName",
		Description: "XName for a compute node",
		Pattern:     `^x(\d{3,5})c(\d{1,3})s(\d{1,3})b(\d{1,3})n(\d{1,3})$`,
	}
}
```

The command has a `schemas` argument that will write out a set of objects to the schemas directory using the reflection features.  See generateAndWriteSchemas for the API to generate jsonschema filles.

```go
func generateAndWriteSchemas(path string) {
	schemas := map[string]interface{}{
		"ComputeNode.json":      &nodes.ComputeNode{},
		"NetworkInterface.json": &nodes.NetworkInterface{},
		"BMC.json":              &nodes.BMC{},
		"Component.json":        &base.Component{},
		"NodeCollection.json":   &nodes.NodeCollection{},
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatal().Err(err).Str("path", path).Msg("Failed to create schema directory")
	}

	for filename, model := range schemas {
		schema := jsonschema.Reflect(model)
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			log.Fatal().Err(err).Str("filename", filename).Msg("Failed to generate JSON schema")
		}
		fullpath := filepath.Join(path, filename)
		if err := os.WriteFile(fullpath, data, 0644); err != nil {
			log.Fatal().Err(err).Str("filename", filename).Msg("Failed to write JSON schema to file")
		}
		log.Info().Str("fullpath", fullpath).Msg("Schema written")
	}
}```


These files are also valuable for clients.  In our python example, the client reads the jsonschema files and can validate a structure on the client side.  In fact, since we can make many assumptions about how to GET and POST these objects, we can create a generic client that doesn't need to understsand these structures directly.

```python
def validate_json(schema_dir, object_type, data):
    if schema_dir:
        schema_path = os.path.join(schema_dir, f"{object_type}.json")
        try:
            with open(schema_path, 'r') as schema_file:
                schema = json.load(schema_file)
            validate(instance=data, schema=schema)
        except FileNotFoundError:
            click.echo(f"Schema file not found: {schema_path}", err=True)
            raise
        except ValidationError as ve:
            click.echo(f"Validation error: {ve}", err=True)
            raise
```

Using it is simple as well.  You create the json object and pass it in.  The client validates it with the schemas in the directory and only passes it to the remote api if everything looks good.  For many use cases, we can test our API calls without needing to start up the remote server.

```python
@cli.command()
@click.argument('object', type=str)
@click.option('--data', type=str, help='JSON string representing the object(s) to create')
@click.option('--file', type=click.File('r'), help='File containing JSON object(s) to create')
def create(object, data, file):
    """Create object(s) on the remote API."""
    if data:
        data = json.loads(data)
    else:
        click.echo("Error: No data provided for creation.", err=True)
        return
    if cli.schema_dir:
        validate_json(cli.schema_dir, object, data)
    response = api_call('POST', cli.url, object, None, data, cli.jwt)
    click.echo(response)
```

## In-Memory Working Set with Periodic Snapshots

Our system employs a pattern that maintains the working set in memory with periodic snapshots to disk.  On startup, the system finds the most recent snapshot and loads it as a working set.  If it cannot find a snapshot, it starts up anyway with an empty working set.

The snapshots themselves use [parquet](https://parquet.apache.org/) files which are column-oriented and optimized for efficient compression, storage, and retrieval.  As a common file format for Big Data, there are dozens of tools that deal natively with the file format in predictable directory structures.

The working set is managed by an in-memory engine called [DuckDB](https://duckdb.org/) which supports efficient SQL and analytical queries on structured data.  The same engine has features that allow it to efficiently execute queries directly against other databases and even directories full of parquet files without needing to load the full dataset into memory.

The duckdb engine can deal very efficiently with parquet files.  Even at inventory sizes of 250K nodes, the snapshot and recovery processes take just a few seconds.

### Customization and Performance
- **Snapshot Frequency**:
  - The sysadmin can configure how often snapshots are taken (e.g., once a minute, once an hour).
  - Frequent snapshots ensure minimal data loss, even in the event of a crash.

- **Snapshot Retention**:
  - The system can be configured to retain a specified number of old snapshots.
  - This allows for rollback to previous states if needed.
  - Queries against historical snapshots allow for offline analytical queries
  - Filesystems that support block-level deduplification improve the performance and reliability of this pattern

#### Technologies Used
1. **DuckDB**:
   - An embedded SQL OLAP database management system that provides fast and efficient data management.
   - Used for in-memory data handling and snapshot operations.
   - Supports direct queries of files on remote storage without loading all data into memory

2. **Parquet**:
   - A columnar storage file format optimized for big data processing.
   - Snapshots are stored in Parquet format, ensuring efficient storage and fast access.
   - Many big data tools natively support Parquet, making the snapshot data easily accessible for analysis.

#### Benefits
- **Speed and Efficiency**:
  - Fast access to data in memory and quick restoration from snapshots.
  - Minimal performance impact from frequent snapshots.

- **Flexibility and Control**:
  - Configurable snapshot frequency and retention.
  - Admins can tune the system to balance between performance and data safety.

- **Big Data Compatibility**:
  - Parquet format snapshots are compatible with many big data tools.
  - Enables advanced data analysis and integration with existing big data workflows.

This pattern ensures a robust, high-performance system that maintains data safety and provides flexibility for administrators. By leveraging DuckDB and Parquet, the system achieves efficient data management and compatibility with big data analysis tools.


## Usage

```bash
go build . && ./node-orchestrator schemas && ./node-orchestrator serve 
python3 -m venv .venv
 .venv/bin/pip install -r clients/requirements.txt
 .venv/bin/python clients/client.py --url http://localhost:8080 --schema-dir schemas create --file client/computenode.json  ComputeNode
 ```

Adjust [computenode.json](/clients/computenode.json) to explore creating and updating different kinds of nodes.
