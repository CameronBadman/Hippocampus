EXTENSION = hippocampus
DATA = hippocampus--1.0.0.sql
MODULES = hippocampus

# PostgreSQL extension build
PG_CONFIG = pg_config
PGXS := $(shell $(PG_CONFIG) --pgxs)
include $(PGXS)

# Additional flags
SHLIB_LINK += -L../bin -lhippocampus
PG_CPPFLAGS += -I../src
