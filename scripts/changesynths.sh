find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"sound.synth\":\".*\",' '\"sound.synth\":\"0103 Ambient_E-Guitar\",'

find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"A-sound.synth\":\".*\",' '\"A-sound.synth\":\"0103 Ambient_E-Guitar\",'
find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"B-sound.synth\":\".*\",' '\"B-sound.synth\":\"0104 Dist_Bass 1\",'
find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"C-sound.synth\":\".*\",' '\"C-sound.synth\":\"0105 Tacky 1\",'
find . -name "*.json" -print0 | xargs -n 1 -0 changesynth1.sh '\"D-sound.synth\":\".*\",' '\"D-sound.synth\":\"0101 Fantasy_Bell\",'

# chall.sh '\"A-sound.synth\":\".*\",' '\"A-sound.synth\":\"0103 Ambient_E-Guitar\",' $@
# chall.sh '\"B-sound.synth\":\".*\",' '\"B-sound.synth\":\"0104 Dist_Bass 1\",' *.json
# chall.sh '\"C-sound.synth\":\".*\",' '\"C-sound.synth\":\"0105 Tacky 1\",' *.json
# chall.sh '\"D-sound.synth\":\".*\",' '\"D-sound.synth\":\"0101 Fantasy_Bell\",' *.json

# chall.sh '\"sound.synth\":\".*\",' '\"sound.synth\":\"0101 Fantasy_Bell\",' *.json
