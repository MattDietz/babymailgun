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
- Don't forget CC and BCC fields
- All 400 email codes are soft bounces and can be retried, all 500 codes are hard and should be skipped on retry
  - Pedantically untrue. https://martech.zone/soft-hard-email-bounce-codes/ Some of these hard bounces could resolve later
- Workers heartbeat in the db periodically. If a worker owns a job and we haven't seen a heartbeat in X, we reclaim
- One worker could create a pool and run hundreds of email jobs in goroutines, maybe do this if it's not too complex
  - At least one goroutine to do heartbeat, no reason we couldn't pick up work and farm it out to goroutines in others



API:
POST /tokens
GET /tokens - Returns token validity
POST /emails {json body of email data model}, returns an ID
GET /emails/<id> - look up the status of an email based on ID

Data Model:

Email
{
  Headers []string,
  Subject string,
  Body string,
  Sender string,
  Recipients: [
    {
      Email string,
      StatusCode int,
      StatusReason string
    }
  ] 
  CreatedAt DateTime,
  UpdatedAt DateTime,
  Status Enum {Incomplete, Complete, Failed}
  StatusReason string <-- The whole email can fail (too many recipients, for example)
  Tries int,
  WorkerId string (uuid), <-- Current owner of this email
}

Worker
{
  HostName string
  IP string
  LastCheckin DateTime
  Checkins int
}

Email: Base model. Tracks the object represented by the API call
EmailRecipient: status tracking model. 

Workflow:
POST /token and get token
POST /emails with token and email body
  -> write to datastore
Workers are watching the database:
  -> Worker wakes up and queries for any Incomplete jobs
    -> SELECT * from emails where worker is NULL and status = "incomplete" and last_updated_at >= 10 minutes ago
  -> Worker picks the top one and atomically assigns itself to it:
    -> UPDATE emails set worker = <id> where last_updated_at = $TIMESTAMP_FROM_FIRST_QUERY and worker = NULL or other db equivalent (mongo findAndModify)
  -> Handshake and send email
  -> For each email (per recipient) that succeeds, update datastore with success
  -> For each email that fails (per recipient basis), update datastore with failure
  -> If there are any failures during sending, update the tries count.
  -> If tries == 3, update status to "Failed"
  -> Always set updated_at to NOW()
  -> Always clear worker field
  # Each ESP has different send rates. Ideally you'd break the emails down by domain and queue those
  -> Go back to sleep for some configurable interval

Email delivery or failure, writing the status back:
  
