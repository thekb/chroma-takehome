# Take home interview
The following is a take home interview task for Chroma. Take home interviews are scoped to a narrow range of time, and demand that candidates make tradeoffs in how they spend their time. We’d like to be up front about what tradeoffs we’d like you to make, based on what we are hoping to learn about your skills and knowledge from this interview.

In particular we are interested in understanding

* Your ability to design intuitable, easy to reason about APIs and turn them into working code
* Your ability to reason about performance, conceptual clarity and generally articulate tradeoffs in a system design.
* Your knowledge of the tools for common problems we expect to face.
* Your personal engineering philosophies

## Task
The open source library SQLite implements a robust SQL database. However, it does not implement a permissioning system. Your task is to design a permissioning system that can be integrated with SQLite.
Given a set of users, an admin should be able to specify the read/write permissions of a given user for a given table, or filtered set of data. For example, a user may only be allowed to read data that they own in a given table or they may only be able to read all data in a given table, or they may be allowed to write all data in a given table. Users should submit their queries and specify their access level using an API key granted to them.

The system should ensure that users can only access data that they have been granted permission to view or modify. The system should be designed in a way that is modular and can be easily integrated with other non-local data sources, in the future. In addition, the system should be flexible enough to allow for different levels of access control, such as read-only access or full read-write access. 

The expected deliverables for this task are:
1. A working local authentication system on top of SQLite in a language of your choice.
2. A brief system design (<1hr) for how you would extend this to a deployed setup to scale
over an existing SQL database - assuming this SQL database also did not support permissioning.

Please complete the above task and submit your solution along with a brief write-up detailing your solution and thought process. Your write-up should touch on the four areas we are interested in understanding: API design, system design tradeoffs, knowledge of relevant tools, and personal engineering philosophies.

We appreciate your time and effort and look forward to reviewing your submission. If you have any questions or concerns, please don't hesitate to reach out to us.
