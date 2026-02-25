---
name: db-query
description: Do query from database for common task
model: sonnet
allowed-tools: Bash(psql *)
---

User could like to know about the status of L2 data blocks and proving task, following is their request:

$ARGUMENTS

(If you find there is nothing in the request above, just tell "nothing to do" and stop)

You should have known the data sheme of our database, if not yet, read it from the `.sql` files under `database/migrate/migrations`.

According to use's request, generate the corresponding SQL expression and query the database. For example, if user ask "list the assigned chunks", it means "query records from `chunk` table with proving_status=2 (assigned)", or the SQL expression 'SELECT * from chunk where proving_status=2;'. If it is not clear, you can ask user which col they are indicating to, and list some possible options.

For the generated SQL, following rules MUST be obey:

+ Limit the number of records to 20, unless user has a specification explicitly like "show me ALL chunks".
+ Following cols can not be read by human and contain very large texts, they MUST be excluded in the SQL expression:
   + For all table, any col named "proof"
   + "header" and "transactions" in `l2_block` table
   + "calldata" in `l1_message`
+ Always omit the `deleted_at` col, never include them in query or use in where condition
+ Without explicit specification, the records should be ordered by the `updated_at` col, the most recent one first.

When you has decided the SQL expression, always print it out.

You use psql client to query from our PostgreSQL db. When launching psql, always with "-w" options, and use "-o" to send all ouput to `query_report.txt` file under system's temporary dir, like /tmp. You MUST NOT read the generated report.

If the psql failed since authentication, remind user to prepare their `.pgpass` file under home dir.

You should have known the endpoint of the database before, in the form of PostgreSQL DSN. If not, try to read it from the `db.dsn` field inside of `coordinator/build/bin/conf/config.json`. If still not able to get the data, ask via Ask User Question to get the endpoint.


