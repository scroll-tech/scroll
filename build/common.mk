UNAME_S := $(shell uname -s)
IS_DARWIN := $(findstring Darwin,$(UNAME_S))

SHLIB_EXT := so
ifeq ($(UNAME_S),Darwin)
SHLIB_EXT := dylib
endif

LIB_ZKP_NAME := libzkp.$(SHLIB_EXT)

define macos_codesign
@if [ -n "$(IS_DARWIN)" ]; then \
  codesign --force --sign - '$(1)'; \
  codesign --verify --deep --verbose '$(1)'; \
fi
endef
