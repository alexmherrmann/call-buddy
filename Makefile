SHELL := /bin/sh
PREFIX := /usr/local
SUBDIRS := telephono telephono-ui

.PHONY: all $(SUBDIRS)
all: $(SUBDIRS)

.PHONY: clean
clean:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d clean; done

telephono:
	$(MAKE) -C telephono all

telephono-ui:
	$(MAKE) -C telephono-ui all

.PHONY: install
install:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d install; done
	@mkdir -p $(PREFIX)/bin
	@cp tcb $(PREFIX)/bin/
	@echo
	@echo "Warning: You need an TCB_ARCH_DIR environment variable pointing to "$(PREFIX)/lib/call-buddy" for the 'tcb' utility to work:"
	@if [ `uname -s` == "Darwin" ]; then \
	    echo "echo 'export TCB_ARCH_DIR=\"$(PREFIX)/lib/call-buddy\"' >> ~/.bash_profile"; \
	else \
	    echo 'export TCB_ARCH_DIR=\"$(PREFIX)/lib/call-buddy\"' >> ~/.bashrc; \
	fi
	@mkdir -p $(PREFIX)/share/man/man1/
	@cp tcb.1 call-buddy.1 $(PREFIX)/share/man/man1/

.PHONY: uninstall
uninstall:
	@for d in $(SUBDIRS); do $(MAKE) -C $$d uninstall; done
	@rm $(PREFIX)/bin/tcb
	@rm $(PREFIX)/share/man/man1/tcb.1 $(PREFIX)/share/man/man1/call-buddy.1

.PHONY: major-release
major-release:
	./release.sh major

.PHONY: minor-release
minor-release:
	./release.sh minor

.PHONY: patch-release
patch-release:
	./release.sh patch
