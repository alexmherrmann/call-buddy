SUBDIRS := telephono telephono-ui launchpad
export prefix="/usr"
export realprefix=$(abspath $(prefix))

.PHONY: all $(SUBDIRS)

all: $(SUBDIRS)

clean:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d clean; done

telephono:
	$(MAKE) -C telephono all

telephono-ui:
	$(MAKE) -C telephono-ui all

launchpad:
	$(MAKE) -C launchpad all

install:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d install; done

uninstall:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d uninstall; done
