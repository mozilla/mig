/*
  Copyright © 2015 Hilko Bengen <bengen@hilluzination.de>. All rights reserved.
  Use of this source code is governed by the license that can be
  found in the LICENSE file.
*/

#include <yara.h>
#include "_cgo_export.h"

int rules_callback(int message, void *message_data, void *user_data) {
  if (message == CALLBACK_MSG_RULE_MATCHING) {
    YR_RULE* rule = (YR_RULE*) message_data;
    char* ns = rule->ns->name;
    if(ns == NULL) {
      ns = "";
    }
    newMatch(user_data, ns, (char*)rule->identifier);
    YR_META* meta;
    yr_rule_metas_foreach(rule, meta) {
      switch (meta->type) {
      case META_TYPE_INTEGER:
        addMetaInt(user_data, (char*)meta->identifier, meta->integer);
        break;
      case META_TYPE_STRING:
        addMetaString(user_data, (char*)meta->identifier, meta->string);
        break;
      case META_TYPE_BOOLEAN:
        addMetaBool(user_data, (char*)meta->identifier, meta->integer);
        break;
      }
    }
    const char* tag_name;
    yr_rule_tags_foreach(rule, tag_name) {
      addTag(user_data, (char*)tag_name);
    }
    YR_STRING* string;
    YR_MATCH* m;
    yr_rule_strings_foreach(rule, string) {
      yr_string_matches_foreach(string, m) {
#if YR_VERSION_HEX >= 0x030500
        /* YR_MATCH members have been renamed in YARA 3.5 */
        addString(user_data, string->identifier, m->offset, m->data, (int)m->data_length);
#else
        addString(user_data, string->identifier, m->offset, m->data, (int)m->length);
#endif
      }
    }
  }
  return CALLBACK_CONTINUE;
}

#ifdef _WIN32
/*
Helper function that is merely used to cast fd from int to HANDLE.
CGO treats HANDLE (void*) to an unsafe.Pointer. This confuses the
go1.4 garbage collector, leading to runtime errors such as:

runtime: garbage collector found invalid heap pointer *(0x5b80ff14+0x4)=0xa0 s=nil
*/
int _yr_rules_scan_fd(
    YR_RULES* rules,
    int fd,
    int flags,
    YR_CALLBACK_FUNC callback,
    void* user_data,
    int timeout)
{
  return yr_rules_scan_fd(rules, (YR_FILE_DESCRIPTOR)fd, flags, callback, user_data, timeout);
}
#endif
