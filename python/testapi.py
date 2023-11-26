import palette

r, err = palette.palette_api("\"api\":\"global.echo\",\"value\":\"THIS IS THE ECHO\"")
print("r=",r," err=",err)
