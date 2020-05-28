# Medchain administration

-------

## CLI calls


### Create

Create a new admin client 

    $ medadmin create

### Admin

     $ medadmin admin subcommand [options] arguments

The admin command manage the admin in the administration darc   

| Subcommand                  | Arguments   | Description |
|:------------------------- |:--------- |:----------- |
| `spawn`                 |           | Spawn a new admin darc |
| `add`             | `--adid -a`, `--identity, -i`      | Add a new admin to admin darc. `adid` the admin darc id. Returns the instance id of the deffered transaction. `identity` the new admin identity string | `--adid -a`, `--identity, -i`     
| `remove`                 |           | Remove an admin from admin darc. Returns the instance id of the deffered transaction. `adid` the admin darc id. `identity` the admin identity string |
| `modify` | `--adid -a`, `--oldkey`, `--newkey`     | Modify the admin key from darc. Returns the instance id of the deffered transaction. `adid` the admin darc id. `oldkey` the old admin identity string. `newkey` the new admin identity string.|

### Deferred

     $ medadmin deffered subcommand [options] arguments

The defferred command manages the deffered transaction registered in the global state of Medchain.  

| Subcommand                   | Arguments   | Description |
|:------------------------- |:--------- |:----------- |
| `sync`                 |           | Get all the instance ID of pending deferred transactions from a Medchain node |
| `get_all`             |  | Returns the list of all pending deffered transactions | `--adid -a`, `--identity, -i`     
| `sign`                 |   `--id -i`        | Sign the deffered transaction. `id` the instance id of the deffered transaction|
| `get` | `--id -i` | Output the content of the deferred transaction. `id` the instance id of the deffered transaction |
| `exec` | `--id -i` | Try to execute the deffered transaction. Only succeed if the signature rule defined in the darc is satisfied. `id` the instance id of the deffered transaction |

### project

     $ medadmin project subcommand [options] arguments

The project command manages the project access rights.  

| Subcommand                  | Arguments   | Description |
|:------------------------- |:--------- |:----------- |
| `create`                 |           | create a new project darc. Returns the instance id of the deffered transaction |
| `accessright`             | `--project_id, --pid`  | Create a new access right contract. Returns the instance id of the deffered transaction. `project_id` the instance id of the project darc  | 
| `attach`                 |   `--access_id -aid`,`--project_id, --pid`        | Attach the access right contract instance id to the project id.`access_id` the instance if of the access right contract. `project_id` the instance id of the project darc |
| `add`                 | `--project_id, --pid`,`--querrier_id, --qid`, `--access`        | Add a new querrier to the project access right contract. Returns the instance id of the deffered transaction. `project_id` the instance id of the project darc. `querrier_id` the id of the querier.  `access` the access rights of the querier |
| `remove`                 | `--project_id, --pid`,`--querrier_id, --qid`       | Removes a querrier from the project access right contract. Returns the instance id of the deffered transaction. `project_id` the instance id of the project darc. `querrier_id` the id of the querier. |
| `modify`                 | `--project_id, --pid`,`--querrier_id, --qid`, `--access`        | Modify the querrier access right. Returns the instance id of the deffered transaction. `project_id` the instance id of the project darc. `querrier_id` the id of the querier.  `access` the new access rights of the querier |
| `verify`                 | `--project_id, --pid`,`--querrier_id, --qid`, `--access`        | Verify the access right of a user. `project_id` the instance id of the project darc. `querrier_id` the id of the querier.  `access` the access right of the querier to verify|
