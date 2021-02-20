from subprocess import Popen, PIPE
from os import path, system


#def start_download(delete, format, url, chapters, bar):
#    path_to_grabber = path.join(path.dirname(__file__), 'grabber.exe')
#    grabber = Popen([path_to_grabber, 
#        f'-url={url}', 
#        f'{"-delete" if delete else ""}',
#        format,
#        chapters], shell=True, stdout=PIPE,)                       #starting grabber

def start_download(delete, format, url, chapters):
    path_to_grabber = path.join(path.dirname(__file__), 'grabber.exe')
    system(f'"{path_to_grabber}" -url={url} {"-delete" if delete else ""} {format} {chapters}')

def connect_buttons(window, settings):
    window.pdf_button.clicked.connect(
        lambda _: settings.update({'format':'-pdf'}))
    window.zip_button.clicked.connect(
        lambda _: settings.update({'format':'-zip'}))
    window.cbz_button.clicked.connect(
        lambda _: settings.update({'format':'-cbz'}))
    window.delete_option.clicked.connect(
        lambda _: settings.update({'delete':not settings['delete']}))

    window.start_button.clicked.connect(
        lambda _: start_download(settings['delete'], 
                                settings['format'], 
                                window.url.text(),
                                window.chapters.text()))
