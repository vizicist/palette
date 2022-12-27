import palette

r, err = palette.palette_api("\"api\":\"engine.echo\",\"value\":\"THIS IS THE ECHO\"")
print("r=",r," err=",err)
