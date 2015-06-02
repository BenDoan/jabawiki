app.factory('ArticleFactory', ["$http", function ArticleFactory($http){
    var exports = {};

    exports.getArticle = function(format, title){
        url = '/article/' + title + '?format=' + format
        return $http({method: 'GET', url: url})
    };

    exports.updateArticle = function(article){
        return $http({
                method: 'put',
                url: '/article/' + article.title,
                data: article
            })
    };

    exports.registerUser = function(){
        return $http({
            method: 'POST',
            url: '/user/register',
            data: {
                email: "testing@bendoan.me",
                name: "Test User",
                password: "password"
            }
        })
    };

    exports.loginUser = function(){
        return $http({
            method: 'POST',
            url: '/user/login',
            data: {
                email: "testing@bendoan.me",
                password: "password"
            }
        })
    };

    return exports;
}]);
