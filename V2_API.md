# Notifications V2 Documentation

- System Status
	- [Check service status](#get-info)
- Senders
	- [Creating a sender](#create-sender)
	- [Retrieving a sender](#retrieve-sender)
- Campaign types
  - [Creating a campaign type](#create-campaign-type)
  - [Showing a campaign type](#show-campaign-type)
  - [Listing the campaign types](#list-campaign-types)
  - [Updating a campaign type](#update-campaign-type)

## System Status

<a name="get-info"></a>
#### Check service status

##### Request

###### Headers
```
X-NOTIFICATIONS-VERSION: 2
```

###### Route
```
GET /info
```

###### CURL example
```
$ curl -i -X GET \
  -H "X-NOTIFICATIONS-VERSION: 2" \
  http://notifications.example.com/info

HTTP/1.1 200 OK
Connection: close
Content-Length: 13
Content-Type: text/plain; charset=utf-8
Date: Tue, 30 Sep 2014 21:29:36 GMT
X-Cf-Requestid: 2cf01258-ccff-41e9-6d82-41a4441af4af

{"version": 2}
```

##### Response

###### Status
```
200 OK
```

###### Body
| Fields  | Description        |
| ------- | ------------------ |
| version | API version number |


## Senders

<a name="create-sender"></a>
#### Creating a sender

##### Request

###### Headers
```
X-NOTIFICATIONS-VERSION: 2
Authorization: Bearer <CLIENT-TOKEN>
```
\* The user token requires `notifications.write` scope.

###### Route
```
POST /senders
```
###### Params

| Key    | Description                               |
| ------ | ----------------------------------------- |
| name\* | the human-readable name given to a sender |

\* required

###### CURL example
```
$ curl -i -X POST \
  -H "X-NOTIFICATIONS-VERSION: 2" \
  -H "Authorization: Bearer <CLIENT-TOKEN>" \
  -d '{"name":"my-sender"}'
  http://notifications.example.com/senders

HTTP/1.1 201 Created
Content-Length: 64
Content-Type: text/plain; charset=utf-8
Date: Fri, 17 Jul 2015 19:30:32 GMT
X-Cf-Requestid: ce9f6b5a-317d-4d0f-7197-df63540c7f22

{"id":"4bbd0431-9f5b-49bb-701d-8c2caa755ed0","name":"my-sender"}
```

##### Response

###### Status
```
201 Created
```

###### Body
| Fields | Description                  |
| ------ | ---------------------------- |
| id     | System-generated sender ID   |
| name   | Sender name                  |

<a name="retrieve-sender"></a>
#### Retrieving a sender

##### Request

###### Headers
```
X-NOTIFICATIONS-VERSION: 2
Authorization: Bearer <CLIENT-TOKEN>
```
\* The user token requires `notifications.write` scope.

###### Route
```
GET /senders/{senderID}
```

###### Params
| Key          | Description                              |
| -------------| ---------------------------------------- |
| senderID\*   | The "id" returned when creating a sender |

\* required

###### CURL Example
```
$ curl -i -X GET \
  -H "Authorization: bearer <CLIENT-TOKEN>" \
  -H "X-NOTIFICATIONS-VERSION: 2" \
  http://notifications.example.com/senders/4bbd0431-9f5b-49bb-701d-8c2caa755ed0

HTTP/1.1 200 OK
Content-Length: 64
Content-Type: text/plain; charset=utf-8
Date: Fri, 17 Jul 2015 21:00:06 GMT
X-Cf-Requestid: 4fab7338-11ba-44d2-75fd-c34046518dae

{"id":"4bbd0431-9f5b-49bb-701d-8c2caa755ed0","name":"my-sender"}
```

##### Response

###### Status
```
200 OK
```

###### Body
| Fields | Description |
| ------ | ----------- |
| id     | Sender ID   |
| name   | Sender name |


## Campaign types

<a name="create-campaign-type"></a>
#### Creating a campaign type

##### Request

###### Headers
```
X-NOTIFICATIONS-VERSION: 2
Authorization: Bearer <CLIENT-TOKEN>
```
\* The user token requires `notifications.write` scope.
\*\* Creation of a critical campaign type requires `critical_notifications.write` scope.

###### Route
```
POST /senders/<sender-id>/campaign-types
```
###### Params

| Key                       | Description                                                         |
| ------------------------- | ------------------------------------------------------------------- |
| name\*                    | the human-readable name given to a campaign type                |
| description\*             | the human-readable description given to a campaign type         |
| critical (default: false) | a flag to indicate whether the campaign type is critical or not |
| template_id               | the ID of a template to use for this campaign type              |

\* required

###### CURL example
```
$ curl -i -X POST \
  -H "X-NOTIFICATIONS-VERSION: 2" \
  -H "Authorization: Bearer <CLIENT-TOKEN>" \
  -d '{"name":"my-campaign-type","description":"campaign type description","critical":false,"template_id":""}'
  http://notifications.mrorange.cfla.cf-app.com/senders/4bbd0431-9f5b-49bb-701d-8c2caa755ed0/campaign_types

HTTP/1.1 201 Created
Content-Length: 155
Content-Type: text/plain; charset=utf-8
Date: Wed, 22 Jul 2015 16:00:37 GMT
X-Cf-Requestid: 6106873b-14ea-4fd9-6418-946c1651e4ac

{"critical":false,"description":"campaign type description","id":"3d9aa963-97bb-4b48-4c3c-ecccad6314f8","name":"my-campaign-type","template_id":""}
```

##### Response

###### Status
```
201 Created
```

###### Body
| Fields        | Description                           |
| ------------- | ------------------------------------- |
| id            | System-generated campaign type ID |
| name          | Campaign type name                |
| description   | Campaign type description         |
| critical      | Critical campaign type flag       |
| template_id   | Template ID                           |

<a name="show-campaign-type"></a>
#### Showing A Campaign type

##### Request

###### Headers
```
X-NOTIFICATIONS-VERSION: 2
Authorization: Bearer <CLIENT-TOKEN>
```
\* The user token requires `notifications.write` scope.

###### Route
```
GET /senders/<sender-id>/campaign-types/<campaign-type-id>
```
###### CURL example
```
$ curl -i -X GET \
  -H "X-NOTIFICATIONS-VERSION: 2" \
  -H "Authorization: Bearer <CLIENT-TOKEN>" \
  http://notifications.example.com/senders/4bbd0431-9f5b-49bb-701d-8c2caa755ed0/campaign_types/3369a6ae-22c5-4da9-7081-b35350c79c4c

200 OK
RESPONSE HEADERS:
  Date: Tue, 28 Jul 2015 00:54:54 GMT
  Content-Length: 155
  Content-Type: text/plain; charset=utf-8
  Connection: close
RESPONSE BODY:
{"critical":false,"description":"campaign type description","id":"3369a6ae-22c5-4da9-7081-b35350c79c4c","name":"my-campaign-type","template_id":""}
```

##### Response

###### Status
```
200 OK
```

###### Body
| Fields             | Description                           |
| ------------------ | ------------------------------------- |
| id                 | System-generated campaign type ID |
| name               | Campaign type name                |
| description        | Campaign type description         |
| critical           | Critical campaign type flag       |
| template_id        | Template ID                           |

<a name="list-campaign-types"></a>
#### Listing Campaign types

##### Request

###### Headers
```
X-NOTIFICATIONS-VERSION: 2
Authorization: Bearer <CLIENT-TOKEN>
```
\* The user token requires `notifications.write` scope.
\*\* Creation of a critical campaign type requires `critical_notifications.write` scope.

###### Route
```
GET /senders/<sender-id>/campaign-types
```
###### CURL example
```
$ curl -i -X GET \
  -H "X-NOTIFICATIONS-VERSION: 2" \
  -H "Authorization: Bearer <CLIENT-TOKEN>" \
  http://notifications.example.com/senders/4bbd0431-9f5b-49bb-701d-8c2caa755ed0/campaign_types

HTTP/1.1 200 OK
Date: Thu, 23 Jul 2015 19:22:46 GMT
Content-Length: 180
Content-Type: text/plain; charset=utf-8

{"campaign_types":[{"critical":false,"description":"campaign type description","id":"702ce4c7-93a0-42b5-4fd5-4d0ed68e2cd7","name":"my-campaign-type","template_id":""}]}
```

##### Response

###### Status
```
200 OK
```

###### Body
| Fields             | Description                           |
| ------------------ | ------------------------------------- |
| campaign_types | The array of campaign types       |
| id                 | System-generated campaign type ID |
| name               | Campaign type name                |
| description        | Campaign type description         |
| critical           | Critical campaign type flag       |
| template_id        | Template ID                          |

<a name="update-campaign-type"></a>
##### Update a Campaign Type

##### Request

###### Headers
```
X-NOTIFICATIONS-VERSION: 2
Authorization: Bearer <CLIENT-TOKEN>
```

\* The user token requires `notifications.write` scope.
\*\* Updating a critical campaign type requires `critical_notifications.write` scope.

###### Route
```
PUT /senders/<sender-id>/campaign_types/<campaign-type-id>
```

###### CURL example
```
$ curl -i -X PUT \
  -H "X-NOTIFICATIONS-VERSION: 2" \
  -H "Authorization: Bearer <CLIENT-TOKEN>" \
  --data '{"name": "new campaign type", "description": "new campaign description", "critical": true}' \
  http://notifications.example.com/senders/a6c38f92-8fa9-488b-4f4c-7f4d4e0c0fd2/campaign_types/5cbc4458-3dba-481b-74c3-4548114b830b

HTTP/1.1 200 OK
Content-Length: 146
Content-Type: text/plain; charset=utf-8
Date: Tue, 04 Aug 2015 20:47:35 GMT

{"critical":true,"description":"new campaign description","id":"5cbc4458-3dba-481b-74c3-4548114b830b","name":"new campaign type","template_id":""}
```

##### Response

###### Status
```
200 OK
```

###### Body
| Fields             | Description                           |
| ------------------ | ------------------------------------- |
| id                 | System-generated campaign type ID |
| name               | Campaign type name                |
| description        | Campaign type description         |
| critical           | Critical campaign type flag       |
| template_id        | Template ID                           |