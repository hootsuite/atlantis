document.addEventListener("DOMContentLoaded", function () {
    var list = document.getElementsByClassName("js-unlock");
    for (var i = 0; i < list.length; i++) {
        list[i].addEventListener('click', function () {
            var unlockUrl = this.getAttribute('data-unlock-url');
            var xhr = new XMLHttpRequest();
            xhr.open('DELETE', unlockUrl);
            xhr.send();
            return xhr;
        });
    }
});
