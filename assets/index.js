var apiURL = window.location.origin + "/api/"

function buildLayout() {
    getFiles(window.location.hash.substr(1));
}

function updateBread() {
    var crumbs = [
        {
            uri: "#",
            name: "Home"
        }
    ];
    var chuncks = window.location.hash.substr(1).split('/');
    var i;
    for(i = 0; i < chuncks.length; i++) {
        var j = 0;
        var uri = "";
        for(j = 0; j < i+1; j++) {
            uri += "/" + chuncks[j];
        }
        var crumb = {
            uri: "#" + uri.substr(1),
            name: decodeURIComponent(chuncks[i])
        };
        crumbs.push(crumb);
    }
    $('.breadcrumb').empty();
    for(i = 0; i < crumbs.length; i++) {
        $('.breadcrumb').append('<a href="'+crumbs[i]['uri']+'" style="color: whitesmoke;">/'+crumbs[i]['name']+'</a>');
    }
}

function getFiles(dir) {
    $('.cards').empty();
    $.getJSON(apiURL + dir)
        .done(function (data) {
            var dirs = [];
            var files = [];
            $.each(data.content, function (id, item) {
                if(item.dir) {
                    dirs.push(item);
                } else {
                    files.push(item);
                }
            });
            $.each(dirs, function (id, item) {
                $('.cards').append('<a href="' + "#" + item.path + '"><article><img href="' + "#" + item.path + '" class="article-img" src="' + item.thumbnail + '" alt=" " /><h1 class="article-title">' + item.name + '</h1></article></a>');
            });
            $.each(files, function (id, item) {
                $('.cards').append('<a target="_blank" href="' + item.path + '"><article><img href="' + "#" + item.path + '" class="article-img" src="' + item.thumbnail + '" alt=" " /><h1 class="article-title">' + item.name + '</h1></article></a>');
            });
            updateBread();
        });
}

$(document).ready(function(){
    buildLayout();
    if ("onhashchange" in window) {
        window.onhashchange = function () {
            getFiles(window.location.hash.substr(1));
        }
    }
});
