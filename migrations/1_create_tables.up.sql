CREATE TABLE pods (
  id SERIAL PRIMARY KEY,
  ts TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  reason VARCHAR(1),
  name varchar,
  namespace varchar,
  selfLink varchar,
  uid varchar,
  resourceVersion varchar,
  creationTimestamp TIMESTAMP WITH TIME ZONE NULL,
  deletionTimestamp TIMESTAMP WITH TIME ZONE NULL,
  nodeName varchar,
  hostIP varchar,
  podIP varchar,
  data json
);

CREATE INDEX pods_name ON pods (name);
CREATE INDEX pods_ts ON pods (ts);
CREATE INDEX pods_namespace ON pods (namespace);
CREATE INDEX pods_podip ON pods (podIP);
CREATE INDEX pods_nodename ON pods (nodeName);

CREATE TABLE nodes (
  id SERIAL PRIMARY KEY,
  ts TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  reason VARCHAR(1),
  name varchar,
  selfLink varchar,
  uid varchar,
  resourceVersion varchar,
  creationTimestamp TIMESTAMP WITH TIME ZONE NULL,
  deletionTimestamp TIMESTAMP WITH TIME ZONE NULL,
  hostIP varchar,
  data json
);

CREATE INDEX nodes_name ON pods (name);
CREATE INDEX nodes_ts ON pods (ts);
CREATE INDEX nodes_hostip ON pods (hostIP);
