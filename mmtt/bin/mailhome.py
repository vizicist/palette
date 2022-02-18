import traceback
import sys
import socket
from email.mime.text import MIMEText
import smtplib, os
from email.MIMEMultipart import MIMEMultipart
from email.MIMEBase import MIMEBase
from email.MIMEText import MIMEText
from email.Utils import COMMASPACE, formatdate
from email import Encoders

def send_email(textfile):
            import smtplib

            gmail_user = "me@timthompson.com"
            gmail_pwd = "mtgdsgldolfepoek"

            try:

                # Create a text/plain message
                msg = MIMEMultipart()

                msg['Subject'] = "Space Palette Log - host="+socket.gethostname() + " file="+textfile
                hostame = socket.gethostname()
                FROM = "me@timthompson.com"
                TO = 'me@timthompson.com'
                msg['From'] = FROM
                msg['To'] = TO

		msg.attach(MIMEText("Space Palette debug log is attached."));

		part = MIMEBase('text', 'plain');
		part.set_payload( open(textfile,"rb").read() )
		Encoders.encode_base64(part)
		part.add_header('Content-Disposition','attachment; filename="%s"' % os.path.basename(textfile))
		msg.attach(part)

                # server = smtplib.SMTP(SERVER) 
                # server = smtplib.SMTP("smtp.gmail.com", 587)
                server = smtplib.SMTP_SSL("smtp.gmail.com", 465)

                server.ehlo()
                # server.starttls()  # needed when using port 587?
                server.login(gmail_user, gmail_pwd)
                server.sendmail(FROM, [TO], msg.as_string())
                #server.quit()
                server.close()
                print 'successfully sent the mail'
            except:
                print "failed to send mail"
                print traceback.format_exc()


if len(sys.argv) < 2:
  print "Usage: %s {file}" % sys.argv[0]
  sys.exit(1)
send_email(sys.argv[1])
