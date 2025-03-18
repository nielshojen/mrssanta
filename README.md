# Mrs Santa

Santa sync server inspired by projects like Rudolph and Moroz

### Goals

* Run on Google Cloud and Azure
* Use serverless components as much as possible
* "Integrate" with Munki, so managed software is only allowed to run if installed by Munki. This is to keep apps patched and be in control of licensing
* Frequent syncs (eg. every 60 secs) to allow close to live blocking/unblocking
* We'll probabnly need a GUI

### What is it

It's a collection of golang and python scripts running in Google Cloud Funtions behind an API Gateway. Data is persisted in a Firebase DB. If a custom domain is preferable, s load balancer is spun up as well.

Theoretically is should scale like crazy without breaking a sweat, but our Mac fleet is fairly small so i couldn't speak to the validity of that. Scaling does happen automatically and that part is completely hands off.

Deployment is all terraform, but a makefile is provided fo make life easier.

Deployment shouldn't take more than 15-20 mins.

### Current status

We have a very basic rule syncing in place. Still need to do management API, more complex rule assignment and Web UI.

Will be looking into making it work on Azure as well at some point

### What's with the stupid name

Well, keeping in the tradition of giving weird and funny name to things in the Mac community, who would tell Santa what to do? Mrs. Santa was the only correct answer!