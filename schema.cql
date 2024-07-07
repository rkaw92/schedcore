CREATE TABLE timers (
    tenant_id uuid,
    timer_id uuid,
    ushard smallint,
    next_at timestamp,
    schedule text,
    done boolean,
    payload text,
    destination text,
    next_invocation_id uuid,
    PRIMARY KEY ((tenant_id, timer_id), ushard)
);

CREATE MATERIALIZED VIEW timers_mat AS
    SELECT next_at, ushard, tenant_id, timer_id, schedule, done, payload, destination, next_invocation_id
    FROM timers
    WHERE next_at IS NOT NULL AND ushard IS NOT NULL AND tenant_id IS NOT NULL AND timer_id IS NOT NULL
    PRIMARY KEY ((ushard, next_at), tenant_id, timer_id);

CREATE TABLE runners (
    ushard smallint,
    next timestamp,
    PRIMARY KEY (ushard)
);

CREATE TABLE invocations (
    tenant_id uuid,
    timer_id uuid,
    invocation_id uuid,
    scheduled_at timestamp,
    real_at timestamp,
    PRIMARY KEY ((tenant_id, timer_id), invocation_id)
);
