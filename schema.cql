CREATE TABLE timers (
    tenant_id uuid,
    timer_id uuid,
    ushard smallint,
    next_at timestamp,
    schedule text,
    enabled boolean,
    done boolean,
    payload text,
    destination text,
    PRIMARY KEY ((tenant_id, timer_id), ushard)
);

CREATE MATERIALIZED VIEW timers_mat AS
    SELECT next_at, ushard, tenant_id, timer_id, schedule, enabled, done, payload, destination
    FROM timers
    WHERE next_at IS NOT NULL AND ushard IS NOT NULL AND tenant_id IS NOT NULL AND timer_id IS NOT NULL
    PRIMARY KEY ((ushard, next_at), tenant_id, timer_id);

CREATE TABLE runners (
    ushard smallint,
    next timestamp,
    PRIMARY KEY (ushard)
);
