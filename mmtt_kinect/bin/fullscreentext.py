from OpenGL.GL import *
from OpenGL.GLUT import *
from OpenGL.GLU import *
from math import *
import glFreeType

window = 0                                             # glut window number
width, height = 500, 400                               # window size
First = True
global font

Cosine = {}
Sine = {}
for angle in range(-5,365,1):
	anglerad = pi * angle / 180.0
	Cosine[angle] = cos(anglerad)
	Sine[angle] = sin(anglerad)

def draw_corner(x, y, radius, ang0, ang1, step=1):
	x0 = x + (1.0 - Sine[ang0]) * radius
	y0 = y - (1.0 - Cosine[ang0]) * radius
	global First
	for angle in range(ang0,ang1,step):
		x1 = x + (1.0 - Sine[angle]) * radius
		y1 = y - (1.0 - Cosine[angle]) * radius
		glVertex2f(x0, y0)
		# print "corner x0 y0 = %f %f" % (x0,y0)
		glVertex2f(x1, y1)
		# print "corner x1 y1 = %f %f" % (x1,y1)
		x0 = x1
		y0 = y1
	First = False

def draw_rect(x, y, width, height, radius, filled = False):

	# print "\ndraw_rect ",x,y,width,height,radius 
	if filled:
		glBegin(GL_POLYGON)
	else:
		glBegin(GL_LINES)

	# left side
	glVertex2f(x, y+radius)
	# print "left   x y = %f %f" % (x,y+radius)
	glVertex2f(x, y+height-radius)
	# print "left   x y = %f %f" % (x,y+radius)

	# upper left corner
	draw_corner(x, y+height, radius, 90, -1, -5)

	# top side
	glVertex2f(x+radius, y+height)
	glVertex2f(x+width-radius, y+height)

	# upper right corner
	draw_corner(x+width-2*radius, y+height, radius, 360, 269, -5)

	# right side
	glVertex2f(x+width, y+height-radius)
	glVertex2f(x+width, y+radius)

	# lower right corner
	draw_corner(x+width-2*radius, y+2*radius, radius, 270, 179, -5)

	# bottom side
	glVertex2f(x+width-radius, y)
	glVertex2f(x+radius, y)

	# lower left corner
	draw_corner(x, y+2*radius, radius, 180, 89, -5)

	glEnd()  

def draw():             # ondraw is called all the time
	glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT) # clear the screen
	glLoadIdentity()                                   # reset position
    
	glColor4f(1.0,0.0,0.0,0.7)

	glEnable(GL_BLEND)
	glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA)

	global font
	font.glPrint(600,480,"Space Palette")
	font.glPrint(600,380,"is initializing")

	# draw_rect(-0.725,-0.175,0.8,0.8,0.2, True)

	glutSwapBuffers()

# initialization
glutInit()                                             # initialize glut
glutInitDisplayMode(GLUT_RGBA | GLUT_DOUBLE | GLUT_ALPHA | GLUT_DEPTH)
glutInitWindowSize(1920, 1080)                      # set window size
glutInitWindowPosition(1024, 0)                           # set window position
window = glutCreateWindow("PaletteProgress")              # create window with title
# glutFullScreen()

fontname = "arial.ttf"
fontheight = 80
font = glFreeType.font_data(fontname,fontheight)

glutDisplayFunc(draw)                                  # set draw function callback
glutIdleFunc(draw)                                     # draw all the time
glutMainLoop()                                         # start everything
