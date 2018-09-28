# Board View for WIT Backend Services

Implements User Stories:
 - Board Configuration on Process Template: https://openshift.io/openshiftio/openshiftio/plan/detail/1725
 - Basic Board View for Planner: https://openshift.io/openshiftio/openshiftio/plan/detail/2073

## Description
This implements the backend support for board views. It contains changes to the JSONAPI services as well as to the space template system_ It provides backend support for a flexible, extensible board view implementation.

## Changes to the JSONAPI

This section describes the API of the new service changes. It defines new entities available through JSONAPI as well as the changes to existing endpoints.

### Board/Column Definition in the Space Template Response

The response for `/spacetemplates` and `/spacetemplates/:id` gets a new relationship for the board definition:

```yaml
# [...]
"workitemboards": {
  "links": {
    "related": "http://api/spacetemplates/000-000-001/workitemboards"
  }
},
# [...]
```

Note that this relationship will *not* be present on the response for `/space/:id` as having normalized template information on the `space` relationships is deprecated. The client should always use the provided `space-template` relationship on the `space` entitiy to pull template information.

The new external relationship `workitemboards` is available at the endpoint `/spacetemplates/:id/workitemboards` and has the following structure: 

```yaml
{
  "id": "000-000-002",
  "attributes": {
    "name": "Scenarios Board",
    "description": "This is the default board config for the legacy template (Experiences).",
    "contextType": "TypeLevelContext",  // this designates the type of the context
    "context": "000-000-003",  // this designates the ID of the context, in this case the typegroup ID
    "created-at": "0001-01-01T00:00:00Z",
    "updated-at": "0001-01-01T00:00:00Z"
  },
  "relationships": {
    "spaceTemplate": {
      "data": {
        "id": "000-000-004",
        "type": "spacetemplates"
      }
    },
    "columns": {
      "data": [
        {
          "id": "000-000-005",
          "type": "boardcolumns"
        }
      ]
    }
  },
  "included": [
    {
      "id": "000-000-005",
      "title": "workitemboardcolumn",
      "columnOrder": 0,  // the left-to-right order of the column in the view
      "type": "boardcolumns"
    }
  ],
  "type": "workitemboards"
}
```

Note that clients should always use the `related` link from the `/spacetemplate` response to pull the column definitions. The column definitions are attached as an included relationship to match the JSONAPI standard.

There might be zero or more `onUpdateActions` defined on a `boardcolumns` definition. The definition, including the `transRuleKey` and `transRuleArguments` values are only used on the WIT side and should not be used by the UI. They are provided for a future use case where the user might be allowed to update those values to change the board behaviour. `transRuleKey` describes the rule/action that is executed when a Work Item is moved *into* this column while `transRuleArguments` gives additional parameters for this action. In the current state of the implementation, this will execute a rule that updates the state of the Work Item moved into the column based on the meta-state given in `transRuleArguments`. In the future, there might be other rules/actions and different arguments. One example could be to create columns for labels. In this case, the rule/action needs to update the labels on the Work Item and the `transRuleArguments` may contain the label attached to the described column.

The columns are described as an embedded JSONAPI relationship. `onUpdateActions` are also described as included relationships.

### Board Position Data on the Work Item Response

The response to `/workitems` and `/workitems/:id` is updated to contain the positions of the Work Item in the boards. Note that a Work Item may have multiple positions as a Work Item can and will appear in multiple boards. A Work Item can appear only once in each board though:

```yaml
{
  "data": [
    {
      "attributes": {
        "system_created_at": "0001-01-01T00:00:00Z",
        "system_title":"Some Work Item",
        # [...]
      },
      relationships: {
         "boardcolumns": {
           "data": [
             {
               "id":"000-000-005",
               "type":"boardcolumns"
             }
           ]
         }  
      },
      # [...]
```

The ID reference of the board column is always referring to the UUID of a column. The board is given implicitly as we're dealing with UUIDs. The column positions are described as an embedded JSONAPI relationship.

### Meta State on the Work Item

The meta-state is not directly connected to the board/column definitions, but to the first application of that system, mapping states to board columns. The meta-state is stored with the Work Item and returned as a standard JSONAPI attribute:

```yaml
{
  "data": [
    {
      "attributes": {
        "system_created_at": "0001-01-01T00:00:00Z",
        "system_title": "Some Work Item",
        "system_metastate": "mInprogress",
        # [...]
      },
# [...]
```

### Rule/Actions on Work Item Data Updates

The rules/actions being executed on Work Item data changes are exposed by a new relationship `onUpdateActions` on the ` /workitemtypes` and `/workitemtypes/:id` responses:

```yaml
# [...]
"onUpdateActions": {
  "data": [
    {
      "id": "000-000-006",
      "type": "onUpdateActions",
      "transRuleKey": "updateColumnPositionFromStateChange",
      "transRuleArguments": { 
        "someArgument": "someValue" 
      }
    }
  ]
}
# [...]
```

There may be zero or more rules defined on a Work Item Type. Note that the rule/action information is not (yet) used on the UI. This might change when we allow the user to update those values to change the board view behaviour. 

The rule/action system might later be used beyond the board view use case as a generic "onUpdate" mechanism.

## Updates on the Board Position and State Values

The above only describes the schema of the board positions and the meta-state values. This section will describe how to update those values.

### Update the Position of Work Items on a Board

When a Work Item is moved on a board by the user, a rule/action is executed on the WIT side on update of the `boardcolumns` relationship. Clients are expected to send a `PATCH` request with the updated list of `boardcolumns` to `/workitems/:id`. *The response of the `PATCH` request contains an updated Work Item that may contain updated attribute and/or relationship values* (for example, an updated `system_state` attribute). The client is expected to update the local Work Item data from that response.

The rule/action being executed on the WIT side is defined by `transRuleID` and `transRuleArguments` values in the board definition (see above). If a card is moved on the board, that rule/action get executed and calculate the updates necessary to the Work Item.

### Update of Work Item Data

When a Work Item is updated (either attributes or relationships), the WIT will re-calculate the board positions of the particular Work Item. The client is expected to send a `PATCH` request to `/workitems/:id` as usual to update Work Item data. *The response of the `PATCH` request contains an updated Work Item that may contain updated attribute and/or relationship values* (for example, an updated `boardcolumns` relationship). The client is expected to update the local Work Item data from that response.

The rule/action being executed on the WIT side is defined by the `onUpdateActions` relationship on the Work Item Type definition (see above). If a Work Item is updated, the rules/actions are executed and the new board positions for the Work Item are calculated.
 
## Querying for Work Items related to a Board View

When displaying the board view, the UI needs a way to get all Work Items that are related to that board view. This is done by extending the query language for criterias on boards and columns:

```
{ $AND: [
    { space: { $EQ: mySpaceID } },
    { typegroup.name: { $EQ: Execution } },
    { iteration: { $EQ: myIterationID } },
    { column: { $EQ: myColumnID }}
  ]}
```

Or for getting all Work Items on an entire board:

```
{ $AND: [
    { space: { $EQ: mySpaceID } },
    { typegroup.name: { $EQ: Execution } },
    { iteration: { $EQ: myIterationID } },
    { board: { $EQ: myBoardID }}
  ]}
```

The new criterias `column` and `board` can be used with other query language features as usual to allow for more fine-granular filtering. The `boardID` and `columnID` values can be retrieved from the board definition contained in the Space Template.

## Internal Wiring

### Default Board Definition in the Space Templates

The default board definition has to be added to the Space Template definition. This is done through the YAML definition files:

```yaml
board_config:

- id: "24181b5c-713f-4bef-a19f-45240875da92"
  name: Scenarios Board
  description: This is the default board config for the legacy template (Scenarios).
  context: "679a563c-ac9b-4478-9f3e-4187f708dd30"
  contextType: "TypeLevelContext"
  columns:
  - id: "b4edad70-1d77-4e5a-b973-0f0d599fd20d"
    title: "New"
    order: 0
    transRuleKey: "updateStateFromColumnMove"
    transRuleArguments: "{ metaState: 'mNew' }"
  - id: "b4edad70-1d77-4e5a-b973-0f0d599fd20d"
    title: "In Progress"
    order: 0
    transRuleKey: "updateStateFromColumnMove"
    transRuleArguments: "{ metaState: 'mInprogress' }"

- id: "56d62801-798a-4bb0-9c97-89f136f3d539"
  name: Experiences Board
  description: This is the default board config for the legacy template (Experiences).
  context: "8e4c995d-8e5f-4ad6-85bd-3cfbe718f908"
  contextType: "TypeLevelContext"
  columns:
  - id: "b4edad70-1d77-4e5a-b973-0f0d599fd20d"
    title: "New"
    order: 0
    transRuleKey: "updateStateFromColumnMove"
    transRuleArguments: "{ metaState: 'mNew' }"
  - id: "ba79f468-d2b3-4f9b-a0de-3817f40d64b4"
    title: "In Progress"
    order: 0
    transRuleKey: "updateStateFromColumnMove"
    transRuleArguments: "{ metaState: 'mInprogress' }"
```

The default board definitions get loaded/updated into the database on launch in the same way the Work Item Type definitions are bootstrapped.

### New Database Table holding the Column Definitions

This is the table that holds the board definition for a specific board of a specific Space. A Work Item placed in a column links back to exactly one `boardcolumn` entry per `boardID`. A Work Item may be in multiple columns but only in one column per board. The `boardcolumns` defintions link back to a Space template (`spaceTemplateID`), not a Space as we don’t allow customization yet and the schema is modelled the same way the Work Item Types are modelled. That means that `columnID` and `boardID` are the same for Spaces using the same Space template. This is not an issue as the Work Item definition links back to the Space, so the context is given.

(Pseudocode)
```sql
CREATE TABLE boardcolumns ( 
   id UUID NOT NULL,
   spaceTemplateID UUID NOT NULL,
   boardID UUID NOT NULL,
   columnID UUID NOT NULL,
   columnOrder INT,
   context UUID NOT NULL,  // typeLevelID for now, but is generic
   contextType VARCHAR NOT NULL, // "TypeLevelContext"
   transRuleKey VARCHAR NOT NULL,  // "updateStateFromColumnMove"
   transRuleArguments JSONB,      // contains { metaState: "mSomeState" }
   columnTitle VARCHAR NOT NULL
)
```

| **ID** | **spaceTemplateID** | **boardID** | **columnID** | **cOrder** | **transRuleID** | **transRuleArguments**       | **context** | **contextType** | **columnTitle** |
| ------ | ------------------- | ----------- | ------------ | ---------- | --------------- | ---------------------------- | ----------- | --------------- | --------------- |
| 1      | s0                  | s0-b0       | s0-b0-c0     | 0          | upFrCMove       | { metaState: "mNew" }        | ExId        | TLContext       | Todo            |
| 2      | s0                  | s0-b0       | s0-b0-c1     | 1          | upFrCMove       | { metaState: "mInprogress" } | ExId        | TLContext       | In Progress     |
| 3      | s0                  | s0-b0       | s0-b0-c2     | 2          | upFrCMove       | { metaState: "mDone }        | ExId        | TLContext       | Done            |
| 4      | s0                  | s0-b1       | s0-b1-c0     | 0          | upFrCMove       | { metaState: "mNew" }        | ScenId      | TLContext       | New             |
| 5      | s0                  | s0-b1       | s0-b1-c1     | 1          | upFrCMove       | { metaState: "mInprogress" } | ScenId      | TLContext       | Working         |
| 6      | s0                  | s0-b1       | s0-b1-c2     | 2          | upFrCMove       | { metaState: "mDone" }       | ScenId      | TLContext       | Closed          |

### Meta-State in the Space Template Definition [DONE]

The meta-state is stored and covered by a new Work Item Type attribute that provides the mapping by using the ordered nature of the enum definitions in the space template. The new attribute is added to the generalized WIT definition so it is inherited by all existing WITs in all templates:

```yaml
"system_metastate":
      label: Meta-State
      description: The meta-state of the work item
      read_only: yes
      required: yes
      type:
        simple_type:
          kind: enum
        base_type:
          kind: string
        # This will allow other WITs to overwrite the values of the state.
        rewritable_values: yes
        # the sequence of the values need to match the sequence of the 
        # system_state attributes. This encapsulates the mapping.
        values: 
        - mNew
        - mOpen
        - mInprogress
        - mResolved
        - mClosed
```

The implementation needs to get the set of `system_state` values and the set of the `system_metastate` values to get the mapping from state to meta-state. The meta-state will be added to the Work Item JSONAPI model response like every other attribute (see above).
