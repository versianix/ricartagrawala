# ricartagrawala
This is an implementation of Ricart-Agrawala algorithm for mutual exclusion on a distributed system.
You can try it using docker and attaching to each process, so make sure you have Docker installed.
On a terminal type:
> docker compose build

Docker will automatically create 4 Images, one for the SharedResource and three for the Processes.
Then you can type:
> docker compose up

Docker will create a container stack with the 4 containers and start. You can then attach to any process, for example:
> docker attach p1

For a process you can type it's ID (1 for the first_process, 2 for the second_process ...) to increment
it's internal logical clock. You can also type x to request to enter the critical section within
the shared resource.

The Ricart-Agrawala is a mutual exclusion algorithm, so, based on the ID of the processes and their 
internal logical clocks, you will never get two different processes accessing the critical
section at the same time!