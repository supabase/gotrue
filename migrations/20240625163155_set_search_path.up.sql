-- set the search_path to an empty string to force fully qualified names in the function 
do $$ 
begin
    -- auth.uid() function
    create or replace function auth.uid() 
        returns uuid 
        set search_path to '' 
    as $func$
        select coalesce(
            nullif(current_setting('request.jwt.claim.sub', true), ''),
            (nullif(current_setting('request.jwt.claims', true), '')::jsonb ->> 'sub')
        )::uuid
    $func$ language sql stable;

    -- auth.role() function
    create or replace function {{ index .Options "Namespace" }}.role() 
        returns text 
        set search_path to ''
    as $func$
        select coalesce(
            nullif(current_setting('request.jwt.claim.role', true), ''),
            (nullif(current_setting('request.jwt.claims', true), '')::jsonb ->> 'role')
        )::text
    $func$ language sql stable;

    -- auth.email() function
    create or replace function {{ index .Options "Namespace" }}.email() 
        returns text 
        set search_path to ''
    as $func$
        select 
        coalesce(
            current_setting('request.jwt.claim.email', true),
            (current_setting('request.jwt.claims', true)::jsonb ->> 'email')
        )::text
    $func$ language sql stable;

    -- auth.jwt() function
    create or replace function {{ index .Options "Namespace" }}.jwt()
        returns jsonb
        set search_path to ''
    as $func$
        select 
        coalesce(
            nullif(current_setting('request.jwt.claim', true), ''),
            nullif(current_setting('request.jwt.claims', true), '')
        )::jsonb;
    $func$ language sql stable;
end $$;
