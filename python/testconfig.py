import sys

import palette

if len(sys.argv) > 1:
    p = sys.argv[1]
else:
    p = "debug"
print(palette.ConfigValue(p))
