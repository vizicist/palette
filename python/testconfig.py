import sys

import spaceutil

if len(sys.argv) > 1:
    p = sys.argv[1]
else:
    p = "debug"
print(spaceutil.ConfigValue(p))
