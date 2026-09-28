# Engineering Decisions


## Go backend

Decision:

Use Go for backend services.


Reason:

The project aims to demonstrate ML engineering and infrastructure skills.

Python will remain focused on machine learning workloads.


Alternative considered:

FastAPI


Chosen:

Go


---


## PostgreSQL storage

Decision:

Use relational storage.


Reason:

ML experiments have structured relationships:

- experiments
- runs
- metrics
- models


Alternative considered:

MongoDB


Chosen:

PostgreSQL