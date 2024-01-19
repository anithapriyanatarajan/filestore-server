from flask import Flask, request, send_from_directory
import os

app = Flask(__name__)
UPLOAD_FOLDER = '/Users/pragya/redhat/test'
app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER

#list files in the folder
@app.route('/')
def list_files():
    return os.listdir(app.config['UPLOAD_FOLDER'])   

@app.route('/upload', methods=['POST'])
def file_upload():
    file = request.files['file']
    if file:
        filename = file.filename
        file.save(os.path.join(app.config['UPLOAD_FOLDER'], filename))
        return 'File uploaded successfully', 200

@app.route('/files/sample.txt')
def download_file(filename):
    return send_from_directory(app.config['UPLOAD_FOLDER'], filename)

if __name__ == '__main__':
    app.run(debug=True)
