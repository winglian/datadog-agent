# Tailers

Tailers are responsible for gathering log messages and sending them for further handling by the remainder of the logs agent.
It is the responsibility of a launcher to create and manage tailers.

A tailer sends log messages via a `chan *message.Message`, and has a unique identifier that can be used as a key.

Tailers are implemented as simple actors, with Start and Stop methods.
