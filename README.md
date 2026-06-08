# advanced-dada-system
a purely vibecoded mess to fan out possible architectures for a terminal software, aimed at linux consultant professionals.
The tool should allow recording and accessing a universal terminal history, both local and remote terminal sessions, enriched with metadata (customer, project, server, etc.).
The history should be nicely formatted, searchable and accessible anywhere.
The software does allow for using LLMs web APIs to do operations such as summarize and ask questions about sessions, projects or customers.
The software should also generate billing reports.

# vibecoding
extensive usage is being made of pure vibecoding to fan out possible architectures for the software, altough this is being publish on github you should absolutely, under no circumstances, use this software outside of a pure dev environment, no promises are made on the quality of the software being produced right now.

# supported architecture
the software is written by Me and for my own requirements, I use a modern fedora desktop with KDE and every design choice takes this into accounts, no other distro or WM or any othervise differente platforms are being considered in the software, if you do come across this and it's helpful for you feel free to contact me.

# development environment
I am developing this software leveraging containerized agents (mainly opencode and antigravity-cli) which gets access to the project repo, the agents has permission to build an rpm package and install it at every release, the agents can then run test on the actual user system with access to the real storage being written by the deployed application, this is in no way safe or required, it's just the choice I made to make iterations over architecture faster, if you do the same I suggest you check the code yourself for bad behaviour, I do. I offer absolutely no guarantees that this will not break your system, your mileage will vary.