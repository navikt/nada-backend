# API specifications

In the API, there are three "roles" a user can have.

- Unauthenticated user
- Authenticated user
    - Has a valid token
- Owner
    - Has a valid token
    - Is part of the team that owns a specific dataproduct

## Endpoints

- Dataproducts
    - [`GET /api/v1/dataproducts`](#get-apiv1dataproducts)
    - [`POST /api/v1/dataproducts`](#post-apiv1dataproducts)
    - [`GET /api/v1/dataproducts/{productID}`](#get-apiv1dataproductsproductid)
    - [`PUT /api/v1/dataproducts/{productID}`](#put-apiv1dataproductsproductid)
    - [`DELETE /api/v1/dataproduct/{productID}`](#delete-apiv1dataproductsproductid)
- Access
    - [`GET /api/v1/access/{productID}`](#get-apiv1accessproductid)
    - [`POST /api/v1/access/{productID}`](#post-apiv1accessproductid)
    - [`DELETE /api/v1/access/{productID}`](#delete-apiv1accessproductid)


### `GET /api/v1/dataproducts`

Retreives a list of registered dataproducts.
Can be accessed by unauthenticated user.

#### Returns

```json
[
    {
        "id": "D1aGrW8PAIQJBaBC2Cel",
        "data_product": {
            "name": "some really cool data",
            "description": "very cool!",
            "datastore":[
                {
                    "bucket_id": "really-cool-data",
                    "project_id": "aura-dev-d9f5",
                    "type": "bucket"
                }
            ],
            "team": "aura",
            "access": {
                "group:aura@nav.no": "0001-01-01T00:00:00Z",
                "user:some.person@nav.no": "2021-06-03T08:05:26Z"
            }
        },
        "updated": "2021-05-26T08:05:31.562305Z",
        "created": "2021-05-25T12:43:14.2823Z"
    }
]
```

### `POST /api/v1/dataproducts`

Registers a dataproduct. 
Can be accessed by an authenticated user.
Requires a JSON object in the request body.

| Fields | Description | Required? | 
|--------|-------------|-----------|
| `name` | The dataproduct's name | Yes |
| `description` | A description of what the dataproduct contains | Yes |
| `team` | The team which owns the dataproduct | Yes |
| `datastore` | A list containing info about the associated datastore. The list must contain at most one element. | No |

The `datastore` field can contain one of two kinds of objects.

##### Bucket

| Fields | Description | Required? |
|--------|-------------|-----------|
| `datastore[0].type` | Must be `"bucket"` | Yes |
| `datastore[0].project_id` | The ID of the Google project where the bucket is found | Yes |
| `datastore[0].bucket_id` | The ID of the bucket | Yes |

##### BigQuery

| Fields | Description | Required? |
|--------|-------------|-----------|
| `datastore[0].type` | Must be `"bigquery"` | Yes |
| `datastore[0].project_id` | The ID of the Google project where the BigQuery resource is found | Yes |
| `datastore[0].dataset_id` | The ID of the BigQuery dataset | Yes |
| `datastore[0].resource_id` | The ID of the BigQuery resource | Yes |
 
#### Returns 

Given the following request body:

```json
{
    "name": "my data",
    "description": "it's my data",
    "team": "my-nais-team",
    "datastore": [
        {
            "type": "bucket",
            "project_id": "my-nais-team-a3e4",
            "bucket_id": "my-bucket"
        }
    ]
}
```

Returns the ID of the newly registered dataproduct:

```
Q8FT4FfQs8nz9agBqvUH
```

### `GET /api/v1/dataproducts/{productID}`

Returns a specific dataproduct.
Can be accessed by an authenticated user.

| Parameters | Description                                   |
|------------|-----------------------------------------------|
| productID  | Must match the ID of a registered dataproduct |

#### Returns

Given `GET /api/v1/dataproducts/Q8FT4FfQs8nz9agBqvUH`

```json
{
    "id": "D1aGrW8PAIQJBaBC2Cel",
    "data_product": {
        "name": "my data",
        "description": "it's my data",
        "datastore":[
            {
                "bucket_id": "my-bucket",
                "project_id": "my-nais-team-a3e4",
                "type": "bucket"
            }
        ],
        "team": "my-nais-team",
        "access": {
            "group:my-nais-team@nav.no": "0001-01-01T00:00:00Z",
        }
    },
    "updated": "2021-05-26T08:05:31.562305Z",
    "created": "2021-05-26T08:05:31.562305Z"
}
```

Notice how the access field contains the owner team with a zero value date.
This means that every team member has read access to the dataproduct forever.


### `PUT /api/v1/dataproducts/{productID}`

Updates fields in the specified productID.
Can be accessed by an owner.

| Parameters | Description                                   |
|------------|-----------------------------------------------|
| productID  | Must match the ID of a registered dataproduct |

Requires a JSON object in the request body. 

| Fields | Description | Required? | 
|--------|-------------|-----------|
| `name` | A new name for the dataproduct | No |
| `description` | A new description | No |
| `team` | A new team, will transfer ownership | No |
| `datastore` | A list containing info about a new associated datastore. The list must contain at most one element. | No |

Only fields that are specified in the request body will be used to update the dataproduct.
Omitted fields will be left unchanged.
Each specified field will be used to overwrite the existing data.

The `datastore` field can contain one of two kinds of objects.

##### Bucket

| Fields | Description | Required? |
|--------|-------------|-----------|
| `datastore[0].type` | Must be `"bucket"` | Yes |
| `datastore[0].project_id` | The ID of the Google project where the bucket is found | Yes |
| `datastore[0].bucket_id` | The ID of the bucket | Yes |

##### BigQuery

| Fields | Description | Required? |
|--------|-------------|-----------|
| `datastore[0].type` | Must be `"bigquery"` | Yes |
| `datastore[0].project_id` | The ID of the Google project where the BigQuery resource is found | Yes |
| `datastore[0].dataset_id` | The ID of the BigQuery dataset | Yes |
| `datastore[0].resource_id` | The ID of the BigQuery resource | Yes |

#### Returns

Given `PUT /api/v1/dataproducts/Q8FT4FfQs8nz9agBqvUH` and the following request body:

```json
{
    "description": "my data isn't that great"
}
```

Returns `204 No Content` if successful. 
Performing another `GET` to view the dataproduct will display the newly made changes.

### `DELETE /api/v1/dataproduct/{productID}`

Deletes the specified dataproduct.
Can be accessed by an owner.

| Parameters | Description                                   |
|------------|-----------------------------------------------|
| productID  | Must match the ID of a registered dataproduct |

#### Returns

Given `DELETE /api/v1/dataproducts/Q8FT4FfQs8nz9agBqvUH`, will return `204 No Content`.
Performing another `GET` to view the dataproduct will return `404 Not Found`.

### `GET /api/v1/access/{productID}`

Retrieves access update-logs for the specified dataproduct.
Can be accessed by an unauthenticated user.

| Parameters | Description                                   |
|------------|-----------------------------------------------|
| productID  | Must match the ID of a registered dataproduct |

#### Returns

Given `GET /api/v1/access/Q8FT4FfQs8nz9agBqvUH`

```json
[
    {
        "dataproduct_id":"Q8FT4FfQs8nz9agBqvUH",
        "author":"some.other.user@nav.no",
        "subject":"some.other.user@nav.no",
        "action":"grant",
        "time":"2021-05-26T08:05:31.85746Z",
        "expires":"2021-06-03T08:05:26Z"
    },
    {
        "dataproduct_id":"Q8FT4FfQs8nz9agBqvUH",
        "author":"some.user@nav.no",
        "subject":"some.user@nav.no",
        "action":"grant",
        "time":"2021-05-25T12:57:13.459317Z",
        "expires":"2021-05-28T12:57:08Z"
    },
    {
        "dataproduct_id":"Q8FT4FfQs8nz9agBqvUH",
        "author":"AccessEnsurance2000",
        "subject":"",
        "action":"verify",
        "time":"2021-05-25T12:56:52.563416Z",
        "expires":"0001-01-01T00:00:00Z"
    }
]
```

The inner elements contain the following fields:

| Field | Description |
|-------|-------------|
| `dataproduct_id` | The dataproduct ID |
| `author` | The user who requested the access update |
| `subject` | The user who is affected by the access update |
| `action` | Either `grant`, `delete` or `verify`. These relate to access being given, taken away or verified |
| `expires` | In case of `"action": "grant"`, the time the access will expire. Otherwise, this has a zero-value time |
| `time` | The time the access update occurred |

### `POST /api/v1/access/{productID}`

Grants access to the specified dataproduct.
Can be accessed by an authenticated user.
Requires a JSON request body.

| Field | Description | Required |
|-------|-------------|----------|
| `subject` | The subject who should be granted access | Yes |
| `type` | The type of subject, can be either `"user"` or `"serviceAccount"` | Yes |
| `expires` | The time the access should be revoked, must be in the future. ISO 8601 datetime formatted string | No |

⚠️ Omitting, or setting a `null` value to `expires` in the request body means that the access will last forever, or until somebody revokes it.

#### Returns

Given `POST /api/v1/access/Q8FT4FfQs8nz9agBqvUH`, with the following JSON request body:

```json
{
    "subject": "yet.another.user@nav.no",
    "type": "user",
    "expires": "2021-06-01T12:00:00Z"
}
```

Returns `204 No Content` if access was successfully given to the subject.
The access update log will reflect the access grant.

### `DELETE /api/v1/access/{productID}`

Revokes access for a subject to the specified dataproduct.
Can be accessed by an owner, or by an authenticated user if they request to revoke their own access.
Requires a JSON request body.

| Field | Description | Required |
|-------|-------------|----------|
| `subject` | The subject who should have their access revoked | Yes |
| `type` | The type of subject, can be either `"user"` or `"serviceAccount"` | Yes |

#### Returns

Given `DELETE /api/v1/access/Q8FT4FfQs8nz9agBqvUH`, with the following JSON request body:

```json
{
    "subject": "yet.another.user@nav.no",
    "type": "user",
}
```

Returns `204 No Content` if access for the subject was successfully revoked.
The access update log will reflect the revoked access.
