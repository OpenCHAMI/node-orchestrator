import click
import requests
import json
import os
import jwt as pyjwt
import datetime
from datetime import timezone
from jsonschema import validate, ValidationError
import asyncio
import aiohttp

def common_options(f):
    f = click.option('--jwt', help='JWT for API authentication', required=False)(f)
    f = click.option('--gen-jwt-secret', help='Secret for generating JWT for testing.', required=False)(f)
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
    if response.status_code in [200, 201, 204, 400]:
        return response
    else:
        response.raise_for_status()

@common_options
@click.group()
def cli(jwt, gen_jwt_secret, schema_dir, url):
    """Command Line Interface for API operations."""
    cli.jwt = jwt
    cli.schema_dir = schema_dir
    cli.url = url
    cli.gen_jwt_secret = gen_jwt_secret
    if cli.gen_jwt_secret:
        cli.jwt =  pyjwt.encode({
            "sub": "alex@lovelltroy.org", # This is the subject of the token, Normally the email address of the user
            "tenant_id": "f8b3b3b7-4b1b-4b7b-8b3b-7b4b1b4b1b4b",
            "tenant_name": "testtenant",
            "tenant_roles": ["user"],
            "partition_id": "f8b3b3b7-4b1b-4b7b-8b3b-7b4b1b4b1b4b",
            "partition_name": "testpartition",
            "partition_roles": ["admin"],
            # These are the standard claims
            "exp": datetime.datetime.now(tz=timezone.utc) + datetime.timedelta(seconds=600),
            "nbf": datetime.datetime.now(tz=timezone.utc),
            "iss": "https://foobar.openchami.cluster",
            "aud": "https://foobar.openchami.cluster",
            "iat": datetime.datetime.now(tz=timezone.utc),
            }, gen_jwt_secret, algorithm="HS256")

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

    async def create_object(obj, semaphore):
        async with semaphore:
            try:
                if cli.schema_dir:
                    validate_json(cli.schema_dir, object, obj)
                response = api_call('POST', cli.url, object, None, obj, cli.jwt)
                if response.status_code not in [200, 201]:
                    if response.status_code == 400:
                        click.echo(f"Error creating object: {obj}, {response.content}")
                    else:
                        click.echo(f" Status Code: {response.status_code}, {response.content}")
            except requests.exceptions.HTTPError as e:
                click.echo(f"Error creating object: {obj}")
                click.echo(e)
            except requests.exceptions.JSONDecodeError as e:
                click.echo(f"Error decoding JSON response when creating object: {obj}")
                click.echo(e)

    async def create_objects():
        semaphore = asyncio.Semaphore(20)
        tasks = []
        if isinstance(data, list):
            for obj in data:
                task = asyncio.create_task(create_object(obj, semaphore))
                tasks.append(task)
        else:
            task = asyncio.create_task(create_object(data, semaphore))
            tasks.append(task)
        await asyncio.gather(*tasks)

    asyncio.run(create_objects())

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

@cli.command()
@click.argument('object', type=str)
@click.argument('id', type=str)
def get(object, id):
    """Get an object from the remote API."""
    response = api_call('GET', cli.url, object, id, None, cli.jwt)
    try:
        click.echo(response.json())
    except requests.exceptions.JSONDecodeError:
        click.echo(response.content)


if __name__ == '__main__':
    cli()

