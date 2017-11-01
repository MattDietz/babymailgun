Babygun


Service Requirements:
- Should provide at least one API method: to send an email.
- Message sending should be retried up to 3 times with 10 minute interval in case of a failure, without keeping API user on the wire.
- All delivery statuses should be logged possibly with some retention policy.

Code Requirements:
- Use Github 
- Please invite @veulkehc, @b0d0nne11, @slava-mg, @horkhe, @jrodom and @jmontemayor to your project
- Minimize the use of frameworks in your application.
- Write close-to-production quality code, whatever this means to you.
- No need to create a deployment but we should be able to run it on our laptops and play with its API
- Please submit your code as a PR so we can easily comment on it.
- During the code review, you are free to make code changes in response to feedback if you feel it is warranted.

Bonus Points:
- Web-based user interface.


Brainstorm:
- Docker-compose
- Customer API endpoint
- Some kind of central store. I like etcd for this
  - Provides distributed store so all the processes in a "region" can use it
  - Plenty of locking semantics for the workers "owning" an email and controlling status
  - Alternatively, I like the idea of "central" services that lock and control everything, and ancillary services that just
    report back to the core service. Downside is there are some scalability issues (but it's all async API, so it's Load Balanceable)
  - Etcd v5
- N workers to send emails (Maybe these could be in Go)
  - These should be blazing fast and lightweight.
  - Clear need for an async piece, as there's an implied retry loop above with a configurable interval
- Mailhog container for receiving emails and vetting they work
- Need a simple SMTP driver to send an email elsewhere too
- Make sure to include tests
- GRPC from workers to service API
- RAML?
- Container build generates a self-signed cert and auth is exposed through SSL
- Auth on the client API
  - JSON Web Tokens for sending an email?
- pyenv lockfile in the basedir
- If implementing API auth, quick bootstrap script that runs with the container build process
- Come up with two plans 1) What I would do with infinite time 2) and what I'm going to implement for the purposes of this


API:
POST /tokens
GET /tokens - Returns token validity
POST /emails {json body of email data model}, returns an ID
GET /emails/<id> - look up the status of an email based on ID


Workflow:

1)
POST /tokens {username: "foo", password: "bar"}
POST /emails Header: Authorization: Bearer $TOKEN {"Subject": "subject", "Body": 
API writes SendEmail to "jobs" key in datastore
One of the N workers picks up the change to the "jobs" key and notes it's a "SendEmail" type
Worker authenticates with an smtp server from the servers key ( there would be an LB in front of these in the real world )



Data Model:
{
  Subject,
  Body,
  Sender,
  Recipient,
  CreatedAt,
  UpdatedAt,
  Status,
  Reason,  <-- Human-readable status field
  Tries,

