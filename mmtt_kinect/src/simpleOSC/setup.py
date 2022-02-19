#!/usr/bin/env python

# please report errors to info@ixi-software.net
# to install :
# python setup.py install

# create a distribution :
# python setup.py sdist

#create win exe installer :
# python setup.py bdist_wininst




from distutils.core import setup





setup(name = 'SimpleOSC',
    version = '0.2.1',
    description = 'SimpleOSC is a simple API for the Open Sound Control for Python ',
    license = 'LGPL',
    author = 'ixi software',
    author_email = 'info@ixi-software.net',
    url = 'http://www.ixi-software.net/backyard',

    packages = ['osc']
)







