# localqueue

A disk FIFO storage tool, used by embedded MQ. It owns segment/checkpoint files
and one directory/name lock. It supplies raw Put/ReadChan/Depth; acknowledgement,
leases, retries and business payload interpretation belong to the MQ service.

Open and OpenWithConfig report initialization errors immediately. NewQueue keeps
the existing source API. Framing and metadata stay unchanged. Close and Delete
release ownership safely; terminal calls can be repeated. Put success means a
frame write, with checkpoint synchronization governed by SyncEvery/SyncTimeout.

Regression tests use disposable directories and cover ownership conflicts,
recovery, durability bounds and concurrent shutdown. No deployment configuration
or consumer service is changed by this tool update.
