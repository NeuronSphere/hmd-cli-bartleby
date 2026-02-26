---
name: document-api
description: Document a NeuronSphere microservice API in RST format
version: "1.0"
author: HMD Labs
requires:
  - hmd-cli-bartleby
tags:
  - api
  - documentation
  - microservice
  - rst
---

# Document API

Generate RST documentation for a NeuronSphere microservice API.

## Instructions

When the user invokes this skill, follow these steps:

### Step 1: Discover the API

1. **Read the service module** at `src/python/hmd_ms_*/hmd_ms_*.py`:
   - Find all `@service.operation` decorators
   - Extract `rest_path`, `rest_methods`, and `args` from each operation
   - Note the operation function names and their docstrings

2. **Read the models** at `src/python/hmd_ms_*/models.py` (or similar):
   - Find Pydantic request and response model classes
   - Note field names, types, defaults, and descriptions
   - Identify which models map to which operations

3. **Read the configuration**:
   - Check `meta-data/config_local.json` for schema loader info and `hmd-lang-*` dependencies
   - Check `meta-data/manifest.json` for `pre_build_artifacts` referencing language packs
   - Note any entity types used by the service

4. **Check for existing docs** in `./docs/`:
   - See if an API reference doc already exists
   - Note the existing toctree structure in `index.rst`

### Step 2: Generate RST Documentation

Create `docs/api_reference.rst` (or an appropriate name) with these sections:

#### Header

```rst
API Reference
=============

This document describes the REST API exposed by <service-name>.
```

#### Endpoints Table

Use a `list-table` directive for a summary of all endpoints:

```rst
.. list-table:: API Endpoints
   :header-rows: 1
   :widths: 30 10 60

   * - Path
     - Method
     - Description
   * - /api/v1/<resource>
     - GET
     - List all <resources>
   * - /api/v1/<resource>
     - POST
     - Create a new <resource>
   * - /api/v1/<resource>/{id}
     - GET
     - Retrieve a <resource> by ID
   * - /api/v1/<resource>/{id}
     - PUT
     - Update a <resource>
   * - /api/v1/<resource>/{id}
     - DELETE
     - Delete a <resource>
```

#### Endpoint Details

For each endpoint, document:

```rst
Create <Resource>
-----------------

.. code-block:: text

   POST /api/v1/<resource>

**Request Body**

.. code-block:: json

   {
     "name": "string",
     "description": "string",
     "config": {}
   }

**Response** (201 Created)

.. code-block:: json

   {
     "identifier": "uuid",
     "name": "string",
     "description": "string",
     "config": {},
     "created_at": "2024-01-01T00:00:00Z"
   }

**Example**

.. code-block:: bash

   curl -X POST http://localhost:8080/api/v1/<resource> \
     -H "Content-Type: application/json" \
     -d '{"name": "example", "description": "An example resource"}'
```

#### Data Models

Document request/response Pydantic models:

```rst
Data Models
-----------

<Resource>
~~~~~~~~~~

.. list-table::
   :header-rows: 1
   :widths: 20 20 10 50

   * - Field
     - Type
     - Required
     - Description
   * - name
     - string
     - Yes
     - The name of the resource
   * - description
     - string
     - No
     - Optional description
   * - config
     - object
     - No
     - Configuration dictionary
```

### Step 3: Update index.rst

Add the new API reference document to the `docs/index.rst` toctree:

```rst
.. toctree::

   install_and_run
   api_reference
   proposals/index
```

### Step 4: Offer to Build

After generating the documentation, offer to build and preview:

```bash
hmd bartleby html
```

The rendered output will be in `./target/bartleby/`.

## Example

### Input: Microservice with user operations

Given a service file with:
```python
@service.operation(rest_path="/api/v1/users", rest_methods=["GET"])
def list_users(args):
    ...

@service.operation(rest_path="/api/v1/users", rest_methods=["POST"])
def create_user(args):
    ...

@service.operation(rest_path="/api/v1/users/{user_id}", rest_methods=["GET"])
def get_user(args):
    ...
```

Generate a complete API reference RST document covering all three endpoints, their request/response shapes based on the models, and curl examples for each.
