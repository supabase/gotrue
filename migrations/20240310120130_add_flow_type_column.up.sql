do $$ begin
   create type flow_type as enum('code', 'pkce');
exception
    when duplicate_object then null;
end $$;

-- TODO: Maybe merge this into a single block. Also check if we need indexes
do $$
begin
   alter table {{ index .Options "Namespace" }}.flow_state
   add column if not exists flow_type flow_type not null default 'code';
end
$$;
