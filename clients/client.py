import click
import requests
import json
import os
from jsonschema import validate, ValidationError

def common_options(f):
    f = click.option('--jwt', help='JWT for API authentication', required=False)(f)
    f = click.option('--schema-dir', type=click.Path(exists=True, file_okay=False, dir_okay=True), required=False, help='Directory containing JSON schema files')(f)
    f = click.option('--url', required=True, help='Base URL for the API')(f)
    return f

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

def api_call(method, url, object, id, data, jwt):
    full_url = f'{url}/{object}'
    if id:
        full_url += f'/{id}'
    headers = {}
    if jwt:
        headers['Authorization'] = f'Bearer {jwt}'
    response = requests.request(method, full_url, json=data, headers=headers)
    if response.status_code in [200, 201, 204]:
        return response.json()
    else:
        response.raise_for_status()

@common_options
@click.group()
def cli(jwt, schema_dir, url):
    """Command Line Interface for API operations."""
    cli.jwt = jwt
    cli.schema_dir = schema_dir
    cli.url = url

@cli.command()
@click.argument('object', type=str)
@click.option('--data', type=str, help='JSON string representing the object(s) to create')
@click.option('--file', type=click.File('r'), help='File containing JSON object(s) to create')
def create(object, data, file):
    """Create object(s) on the remote API."""
    if file:
        data = json.load(file)
    elif data:
        data = json.loads(data)
    else:
        click.echo("Error: No data provided for creation.", err=True)
        return
    if isinstance(data, list):
        print(data)
        for obj in data:
            if cli.schema_dir:
                validate_json(cli.schema_dir, object, obj)
            try:
                response = api_call('POST', cli.url, object, None, obj, cli.jwt)
                click.echo(response)
            except requests.exceptions.HTTPError as e:
                click.echo(f"Error creating object: {obj}")
                click.echo(e)
    else:
        if cli.schema_dir:
            validate_json(cli.schema_dir, object, data)
        response = api_call('POST', cli.url, object, None, data, cli.jwt)
        click.echo(response)

@cli.command()
@click.argument('object', type=str)
@click.argument('id', type=str)
@click.option('--data', type=str, help='JSON string representing the object to update')
@click.option('--file', type=click.File('r'), help='File containing JSON object to update')
def update(object, id, data, file):
    """Update an object on the remote API."""
    if file:
        data = json.load(file)
    elif data:
        data = json.loads(data)
    else:
        click.echo("Error: No data provided for update.", err=True)
        return
    if cli.schema_dir:
        validate_json(cli.schema_dir, object, data)
    response = api_call('PUT', cli.url, object, id, data, cli.jwt)
    click.echo(response)

@cli.command()
@click.argument('object', type=str)
@click.argument('id', type=str)
def delete(object, id):
    """Delete an object from the remote API."""
    response = api_call('DELETE', cli.url, object, id, None, cli.jwt)
    click.echo(f'Deleted {object} with ID {id}')

if __name__ == '__main__':
    cli()

