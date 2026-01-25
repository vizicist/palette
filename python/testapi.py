import palette

r, err = palette.palette_api("global.get","\"name\":\"global.guisize\"")
print("r=",r," err=",err)
