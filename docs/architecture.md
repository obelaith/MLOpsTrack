# MLFlowOps Architecture


## Overview

MLFlowOps is separated into three main components:

1. Backend services
2. Machine learning workflows
3. User interface


## Backend

Technology:
Go

Responsibilities:

- API handling
- Authentication
- Experiment management
- Job scheduling
- Model registry


The backend does not train models.


## ML Layer

Technology:
Python + PyTorch

Responsibilities:

- Model training
- Evaluation
- Metric reporting
- Artifact creation


## Frontend

Technology:
React

Responsibilities:

- Display experiments
- Compare runs
- Manage models


## Data Storage

PostgreSQL:

Stores:

- experiments
- runs
- metrics
- models


Redis:

Used for:

- job queues
- temporary state


## Deployment

Development:

Docker Compose


Production:

Cloud deployment using containerized services.