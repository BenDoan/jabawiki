app.factory('ArticleFactory', ["$http", function ArticleFactory($http){
    var exports = {};

    exports.getArticle = function(format, title){
        url = '/article?format=' + format + '&title=' + title
        return $http({method: 'GET', url: url})
    };

    exports.updateArticle = function(article){
        return $http({
                method: 'put',
                url: '/article',
                data: article
            })
    };

    exports.registerUser = function(){
        return $http({
            method: 'POST',
            url: '/user/register',
            data: "email=test@bendoan.me&name=Ben Doan&password=pass",
            headers: {'Content-Type': 'application/x-www-form-urlencoded'}
        })
    };

    return exports;
}]);
