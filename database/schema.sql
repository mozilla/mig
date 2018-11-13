CREATE TABLE actions (
    id              numeric NOT NULL,
    name            character varying(2048) NOT NULL,
    target          character varying(2048) NOT NULL,
    description     json,
    threat          json,
    operations      json,
    validfrom       timestamp with time zone NOT NULL,
    expireafter     timestamp with time zone NOT NULL,
    starttime       timestamp with time zone,
    finishtime      timestamp with time zone,
    lastupdatetime  timestamp with time zone,
    status          character varying(256),
    syntaxversion   integer,
    pgpsignatures   character varying(4096) NOT NULL
);
ALTER TABLE public.actions OWNER TO migadmin;
ALTER TABLE ONLY actions
    ADD CONSTRAINT actions_pkey PRIMARY KEY (id);

CREATE TABLE agents (
    id                  numeric NOT NULL,
    name                character varying(2048) NOT NULL,
    queueloc            character varying(2048) NOT NULL,
    mode                character varying(2048) NOT NULL,
    version             character varying(2048) NOT NULL,
    pid                 integer NOT NULL,
    starttime           timestamp with time zone NOT NULL,
    destructiontime     timestamp with time zone,
    heartbeattime       timestamp with time zone NOT NULL,
    refreshtime         timestamp with time zone NOT NULL,
    status              character varying(255),
    environment         json,
    tags                json,
    loadername          character varying(2048)
);
ALTER TABLE public.agents OWNER TO migadmin;
ALTER TABLE ONLY agents
    ADD CONSTRAINT agents_pkey PRIMARY KEY (id);
CREATE INDEX agents_heartbeattime_idx ON agents(heartbeattime DESC);
CREATE INDEX agents_starttime_idx ON agents(starttime DESC);
CREATE INDEX agents_queueloc_pid_idx ON agents(queueloc, pid);
CREATE INDEX agents_status_idx ON agents(status);

CREATE TABLE agents_stats (
    timestamp                   timestamp with time zone not null,
    online_agents               numeric,
    online_agents_by_version    json,
    online_endpoints            numeric,
    idle_agents                 numeric,
    idle_agents_by_version      json,
    idle_endpoints              numeric,
    new_endpoints               numeric,
    multi_agents_endpoints      numeric,
    disappeared_endpoints       numeric,
    flapping_endpoints          numeric
);

CREATE TABLE agtmodreq (
    moduleid        numeric NOT NULL,
    agentid         numeric NOT NULL,
    minimumweight   integer NOT NULL
);
ALTER TABLE public.agtmodreq OWNER TO migadmin;
CREATE UNIQUE INDEX agtmodreq_moduleid_agentid_idx ON agtmodreq USING btree (moduleid, agentid);
CREATE INDEX agtmodreq_agentid_idx ON agtmodreq USING btree (agentid);
CREATE INDEX agtmodreq_moduleid_idx ON agtmodreq USING btree (moduleid);

CREATE TABLE commands (
    id          numeric NOT NULL,
    actionid    numeric NOT NULL,
    agentid     numeric NOT NULL,
    status      character varying(255) NOT NULL,
    results     json,
    starttime   timestamp with time zone NOT NULL,
    finishtime  timestamp with time zone
);
ALTER TABLE public.commands OWNER TO migadmin;
ALTER TABLE ONLY commands
    ADD CONSTRAINT commands_pkey PRIMARY KEY (id);
CREATE INDEX commands_agentid ON commands(agentid DESC);
CREATE INDEX commands_actionid ON commands(actionid DESC);

CREATE TABLE invagtmodperm (
    investigatorid  numeric NOT NULL,
    agentid         numeric NOT NULL,
    moduleid        numeric NOT NULL,
    weight          integer NOT NULL
);
ALTER TABLE public.invagtmodperm OWNER TO migadmin;
CREATE UNIQUE INDEX invagtmodperm_investigatorid_agentid_moduleid_idx ON invagtmodperm USING btree (investigatorid, agentid, moduleid);
CREATE INDEX invagtmodperm_agentid_idx ON invagtmodperm USING btree (agentid);
CREATE INDEX invagtmodperm_investigatorid_idx ON invagtmodperm USING btree (investigatorid);
CREATE INDEX invagtmodperm_moduleid_idx ON invagtmodperm USING btree (moduleid);

CREATE SEQUENCE investigators_id_seq START 1;
CREATE TABLE investigators (
    id              numeric NOT NULL DEFAULT nextval('investigators_id_seq'),
    name            character varying(1024) NOT NULL,
    pgpfingerprint  character varying(128),
    publickey       bytea,
    privatekey      bytea,
    status          character varying(255) NOT NULL,
    createdat       timestamp with time zone NOT NULL,
    lastmodified    timestamp with time zone NOT NULL,
    permissions     bigint NOT NULL DEFAULT 0,
    apikey          bytea,
    apisalt         bytea
);
ALTER TABLE public.investigators OWNER TO migadmin;
ALTER TABLE ONLY investigators
    ADD CONSTRAINT investigators_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX investigators_pgpfingerprint_idx ON investigators USING btree (pgpfingerprint);

CREATE SEQUENCE manifests_id_seq START 1;
CREATE TABLE manifests (
	id        numeric NOT NULL DEFAULT nextval('manifests_id_seq'),
	name      character varying(256) NOT NULL,
	content   text NOT NULL,
	timestamp timestamp with time zone NOT NULL,
	status    character varying(255) NOT NULL,
	target    character varying(2048) NOT NULL
);
ALTER TABLE public.manifests OWNER TO migadmin;
ALTER TABLE ONLY manifests
    ADD CONSTRAINT manifests_pkey PRIMARY KEY (id);

CREATE TABLE manifestsig (
	manifestid     numeric NOT NULL,
	investigatorid numeric NOT NULL,
	pgpsignature   character varying(4096) NOT NULL
);
CREATE UNIQUE INDEX manifestsig_manifestid_investigatorid_idx ON manifestsig USING btree(manifestid, investigatorid);

CREATE SEQUENCE loaders_id_seq START 1;
CREATE TABLE loaders (
	id            numeric NOT NULL DEFAULT nextval('loaders_id_seq'),
	loadername    character varying(256) NOT NULL,
	keyprefix     character varying(256) NOT NULL,
	loaderkey     bytea NOT NULL,
	salt          bytea NOT NULL,
	name          character varying(2048),
	env           json,
	tags          json,
	lastseen      timestamp with time zone NOT NULL,
	enabled       boolean NOT NULL DEFAULT false,
	expectenv     character varying(2048),
	queueloc      character varying(2048)
);
ALTER TABLE ONLY loaders
    ADD CONSTRAINT loaders_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX loaders_loadername_idx ON loaders USING btree(loadername);
CREATE UNIQUE INDEX loaders_loaderkey_idx ON loaders USING btree(loaderkey);
CREATE UNIQUE INDEX loaders_keyprefix_idx ON loaders USING btree(keyprefix);
CREATE UNIQUE INDEX loaders_queueloc_idx ON loaders USING btree(queueloc);
ALTER TABLE public.loaders OWNER TO migadmin;

CREATE TABLE modules (
    id      numeric NOT NULL,
    name    character varying(256) NOT NULL
);
ALTER TABLE public.modules OWNER TO migadmin;
ALTER TABLE ONLY modules
    ADD CONSTRAINT modules_pkey PRIMARY KEY (id);

CREATE TABLE signatures (
    actionid        numeric NOT NULL,
    investigatorid  numeric NOT NULL,
    pgpsignature    character varying(4096) NOT NULL
);
ALTER TABLE public.signatures OWNER TO migadmin;
CREATE UNIQUE INDEX signatures_actionid_investigatorid_idx ON signatures USING btree (actionid, investigatorid);
CREATE INDEX signatures_actionid_idx ON signatures USING btree (actionid);
CREATE INDEX signatures_investigatorid_idx ON signatures USING btree (investigatorid);

ALTER TABLE ONLY agtmodreq
    ADD CONSTRAINT agtmodreq_moduleid_fkey FOREIGN KEY (moduleid) REFERENCES modules(id);

ALTER TABLE ONLY commands
    ADD CONSTRAINT commands_actionid_fkey FOREIGN KEY (actionid) REFERENCES actions(id);

ALTER TABLE ONLY commands
    ADD CONSTRAINT commands_agentid_fkey FOREIGN KEY (agentid) REFERENCES agents(id);

ALTER TABLE ONLY manifestsig
    ADD CONSTRAINT manifestsig_manifestid_fkey FOREIGN KEY (manifestid) REFERENCES manifests(id);

ALTER TABLE ONLY invagtmodperm
    ADD CONSTRAINT invagtmodperm_agentid_fkey FOREIGN KEY (agentid) REFERENCES agents(id);

ALTER TABLE ONLY invagtmodperm
    ADD CONSTRAINT invagtmodperm_investigatorid_fkey FOREIGN KEY (investigatorid) REFERENCES investigators(id);

ALTER TABLE ONLY invagtmodperm
    ADD CONSTRAINT invagtmodperm_moduleid_fkey FOREIGN KEY (moduleid) REFERENCES modules(id);

ALTER TABLE ONLY signatures
    ADD CONSTRAINT signatures_actionid_fkey FOREIGN KEY (actionid) REFERENCES actions(id);

ALTER TABLE ONLY signatures
    ADD CONSTRAINT signatures_investigatorid_fkey FOREIGN KEY (investigatorid) REFERENCES investigators(id);

-- Scheduler can read all tables, insert and select private keys in the investigators table, but cannot update investigators
GRANT SELECT ON ALL TABLES IN SCHEMA public TO migscheduler;
GRANT INSERT, UPDATE ON actions, commands, agents, agents_stats, signatures TO migscheduler;
GRANT INSERT ON investigators TO migscheduler;
GRANT USAGE ON SEQUENCE investigators_id_seq TO migscheduler;

-- API has limited permissions, and cannot list scheduler private keys in the investigators table, but can update their statuses
GRANT SELECT ON actions, agents, agents_stats, agtmodreq, commands, invagtmodperm, loaders, manifests, manifestsig, modules, signatures TO migapi;
GRANT SELECT (id, name, pgpfingerprint, publickey, status, createdat, lastmodified, permissions, apikey, apisalt) ON investigators TO migapi;
GRANT INSERT ON agents, actions, signatures, manifests, manifestsig, loaders TO migapi;
GRANT DELETE ON manifestsig TO migapi;
GRANT INSERT (name, pgpfingerprint, publickey, status, createdat, lastmodified, permissions) ON investigators TO migapi;
GRANT UPDATE (permissions, status, lastmodified, apikey, apisalt) ON investigators TO migapi;
GRANT UPDATE (name, env, tags, loaderkey, salt, lastseen, enabled, expectenv, queueloc) ON loaders TO migapi;
GRANT UPDATE (status) ON manifests TO migapi;
GRANT USAGE ON SEQUENCE investigators_id_seq TO migapi;
GRANT USAGE ON SEQUENCE loaders_id_seq TO migapi;
GRANT USAGE ON SEQUENCE manifests_id_seq TO migapi;

-- readonly user is used for things like expanding targets
CREATE ROLE migreadonly;
ALTER ROLE migreadonly WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB NOLOGIN;
GRANT SELECT ON actions, agents, agtmodreq, commands, invagtmodperm, modules, signatures TO migreadonly;
GRANT SELECT (id, env, tags, expectenv, loadername, queueloc) ON loaders TO migreadonly;
GRANT SELECT (id, name, pgpfingerprint, publickey, status, createdat, lastmodified) ON investigators TO migreadonly;
GRANT migreadonly TO migapi;
GRANT migreadonly TO migscheduler;
