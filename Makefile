SUBDIRS := telephono telephono-ui
export prefix="/usr/local"
export realprefix=$(abspath $(prefix))

.PHONY: all $(SUBDIRS)

all: $(SUBDIRS)

clean:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d clean; done

telephono:
	$(MAKE) -C telephono all

telephono-ui:
	$(MAKE) -C telephono-ui all

install:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d install; done
	@mkdir -p $(realprefix)/bin
	@cp tcb $(realprefix)/bin/
	@echo
	@echo "Warning: You need an TCB_ARCH_DIR environment variable pointing to "$(realprefix)/lib/call-buddy" for the 'tcb' utility to work:"
	@if [ `uname -s` == "Darwin" ]; then \
	    echo "echo 'export TCB_ARCH_DIR=\"$(realprefix)/lib/call-buddy\"' >> ~/.bash_profile"; \
	else \
	    echo 'export TCB_ARCH_DIR=\"$(realprefix)/lib/call-buddy\"' >> ~/.bashrc; \
	fi

uninstall:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d uninstall; done
