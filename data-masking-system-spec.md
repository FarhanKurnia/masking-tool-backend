
# Data Masking Tool — Full System Specification
AI-Ready Specification for Fullstack Implementation

This document is designed so an AI coding agent (GitHub Copilot / Copilot Chat / Cursor / similar) can generate a fullstack application.

System Stack
- Frontend: React + Vite + MUI
- Backend: Node.js + Express
- Database: PostgreSQL

---

# 1. Business Goal

Companies often need production data for:

- testing
- analytics
- QA
- staging environments

However production data contains sensitive information such as:

- emails
- phone numbers
- names
- addresses
- financial identifiers

This system allows users to:

1. Connect to a production database
2. Select tables
3. Configure masking rules
4. Run a masking job
5. Export sanitized dataset

---

# 2. High Level Architecture

User → React UI → REST API → Job Processor → Database → Masking Engine → Export

Components:

Frontend (React + Vite + MUI)
Backend API (Node.js + Express)
Masking Engine
PostgreSQL Database

---

# 3. Wizard Flow

Step 1 — Connect Production DB  
Step 2 — Select Table  
Step 3 — Configure Masking  
Step 4 — Configure Output  
Step 5 — Run Job

---

# 4. Flowchart

User
↓
Create Job Wizard
↓
Connect DB
↓
Fetch Tables
↓
Select Table
↓
Detect Columns
↓
Configure Masking
↓
Configure Output
↓
Submit Job
↓
Backend Create Job
↓
Job Processor Execute
↓
Apply Masking
↓
Generate Output
↓
Job Completed

---

# 5. Masking Types

Supported masking strategies

| Mask Type | Description |
|-----------|-------------|
| full | Replace entire value |
| partial | Keep part |
| email | Mask email |
| random_string | Random string |
| random_number | Random number |
| null | Replace with NULL |
| hash | SHA256 hash |

Example:

email
john.doe@gmail.com → j***@gmail.com

phone
0812345678 → 08******78

---

# 6. Database Design

Tables:

jobs
job_tables
masking_rules
job_runs

---

## jobs

id UUID PK  
name VARCHAR  
source_db_host VARCHAR  
source_db_port INT  
source_db_name VARCHAR  
source_db_user VARCHAR  
source_db_password VARCHAR  
output_type VARCHAR  
created_at TIMESTAMP  

---

## job_tables

id UUID PK  
job_id UUID FK → jobs.id  
table_name VARCHAR  

---

## masking_rules

id UUID PK  
job_id UUID FK → jobs.id  
column_name VARCHAR  
mask_type VARCHAR  
parameters JSON  

---

## job_runs

id UUID PK  
job_id UUID FK → jobs.id  
status VARCHAR  
rows_processed INT  
started_at TIMESTAMP  
finished_at TIMESTAMP  
log TEXT  

---

# 7. ER Relationship

jobs
├── job_tables
├── masking_rules
└── job_runs

jobs.id is parent key

---

# 8. API Contract

Base URL

/api/v1

All responses JSON

---

## Create Job

POST /jobs

Request

{
"name": "Mask Customer Table",
"sourceDB": {
"host": "localhost",
"port": 5432,
"database": "production",
"user": "admin",
"password": "secret"
},
"table": "customers",
"output": "csv"
}

Response

{
"id": "job_uuid",
"status": "created"
}

---

## List Jobs

GET /jobs

Response

[
{
"id": "uuid",
"name": "Mask Customers",
"status": "completed",
"created_at": "..."
}
]

---

## Save Masking Rules

POST /jobs/{id}/masking-rules

Request

{
"rules": [
{
"column": "email",
"type": "email"
},
{
"column": "phone",
"type": "partial"
}
]
}

---

## Run Job

POST /jobs/{id}/run

Response

{
"run_id": "uuid",
"status": "running"
}

---

## Job Status

GET /runs/{run_id}

Response

{
"status": "completed",
"rows_processed": 120000
}

---

# 9. Frontend Integration

Frontend uses these endpoints:

GET /jobs  
POST /jobs  
POST /jobs/{id}/masking-rules  
POST /jobs/{id}/run  
GET /runs/{run_id}

React pages:

Dashboard.jsx → list jobs

NewJob.jsx → wizard

Steps

StepConnectDB.jsx
StepSelectTable.jsx
StepMaskingConfig.jsx
StepOutputConfig.jsx
StepRunJob.jsx

API service file

src/api/jobApi.js

Functions

createJob()
listJobs()
saveMaskingRules()
runJob()
getRunStatus()

