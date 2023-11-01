package handlers

import "github.com/gofiber/fiber/v2"

func Form(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

const html = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Simple HTML Form</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/materialize/1.0.0/css/materialize.min.css">
    <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
</head>
    <style>
        body {
            background-color: #ffffff;
        }

        header h1 {
            text-transform: uppercase;
            font-size: 70px;
            font-weight: 600;
            color: #fdfdfe;
            text-shadow: 0px 0px 5px #b393d3, 0px 0px 10px #b393d3, 0px 0px 10px #b393d3,
                0px 0px 20px #b393d3;
        }
        .logo-title {
            font-family: 'Arial', sans-serif;
            font-size: 2rem;
            color: #fff;
            margin-bottom: 20px;
        }
        .logo {
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <!--
        <div class="logo">
            <img src="logo.png" alt="Logo">
        </div>
        -->
        <header>
            <h1 class="center-align logo-title">ladddddddder</h1>
        </header>
        <form id="inputForm" class="col s12" method="get">
            <div class="row">
                <div class="input-field col s10">
                    <input type="text" id="inputField" name="inputField" class="validate" required>
                    <label for="inputField">URL</label>
                </div>
                <div class="input-field col s2">
                    <button class="btn waves-effect waves-light" type="submit" name="action">Submit
                        <i class="material-icons right">send</i>
                    </button>
                </div>
            </div>
            <div class="row center-align">
            </div>
        </form>
    </div>

    <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/materialize/1.0.0/js/materialize.min.js"></script>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            M.AutoInit();
        });
        document.getElementById('inputForm').addEventListener('submit', function (e) {
            e.preventDefault();
            const inputValue = document.getElementById('inputField').value;
            window.location.href = '/' + inputValue;
            return false;
        });
    </script>
</body>
</html>
`
