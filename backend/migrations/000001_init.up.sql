CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    name          TEXT NOT NULL,
    avatar_url    TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE workspaces (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    slug          TEXT UNIQUE NOT NULL,
    created_by    UUID REFERENCES users(id),
    deadline_day  INT CHECK(deadline_day BETWEEN 0 AND 6),
    deadline_hour INT CHECK(deadline_hour BETWEEN 0 AND 23),
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE workspace_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE CASCADE,
    role         TEXT NOT NULL CHECK(role IN ('admin','member','viewer')),
    invited_by   UUID REFERENCES users(id),
    joined_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, user_id)
);

CREATE TABLE invitations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    email        TEXT NOT NULL,
    token        TEXT UNIQUE NOT NULL,
    expires_at   TIMESTAMP NOT NULL,
    accepted_at  TIMESTAMP
);

CREATE TABLE weekly_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    week_start_date DATE NOT NULL,
    done            TEXT,
    blockers        TEXT,
    next_week       TEXT,
    score           INT CHECK(score BETWEEN 1 AND 5),
    submitted_at    TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, user_id, week_start_date)
);

CREATE TABLE report_summaries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    week_start_date DATE NOT NULL,
    summary_text    TEXT NOT NULL,
    generated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    model           TEXT,
    UNIQUE(workspace_id, week_start_date)
);

CREATE TABLE workspace_ai_settings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE UNIQUE,
    provider     TEXT NOT NULL CHECK(provider IN ('openai','anthropic','gemini')),
    api_key_enc  TEXT NOT NULL,
    model        TEXT,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE todoist_integrations (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    access_token   TEXT NOT NULL,
    last_synced_at TIMESTAMP,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE todoist_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    todoist_task_id TEXT NOT NULL,
    content         TEXT NOT NULL,
    project_name    TEXT,
    due_date        DATE,
    completed_at    TIMESTAMP,
    week_start_date DATE,
    fetched_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, todoist_task_id)
);
