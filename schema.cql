CREATE TABLE timers_64 (
    tenant_id uuid,
    timer_id uuid,
    ushard smallint,
    next_at timestamp,
    schedule text,
    enabled boolean,
    done boolean,
    PRIMARY KEY ((tenant_id, timer_id), ushard)
);

CREATE MATERIALIZED VIEW timers_64_mat AS
    SELECT next_at, ushard, tenant_id, timer_id, schedule, enabled, done
    FROM timers_64
    WHERE next_at IS NOT NULL AND ushard IS NOT NULL AND tenant_id IS NOT NULL AND timer_id IS NOT NULL
    PRIMARY KEY ((ushard, next_at), tenant_id, timer_id);
