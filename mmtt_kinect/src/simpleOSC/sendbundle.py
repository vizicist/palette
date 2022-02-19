import osc

osc.init()
    
# create and send a bundle
bundle = osc.createBundle()
osc.appendToBundle(bundle, "/test/bndlprt1", [1, 2.2, "333"])
osc.appendToBundle(bundle, "/test/bndlprt2", [4, 5.5, 6])
osc.sendBundle(bundle, "127.0.0.1", 9999)
