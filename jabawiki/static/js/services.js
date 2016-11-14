app.factory('ArticleFactory', ["$http", "$httpParamSerializerJQLike", function ArticleFactory($http, $httpParamSerializerJQLike){
    var exports = {};

    exports.getArticle = function(title){
        url = '/article/get/' + title
        return $http({method: 'GET', url: url})
    };

    exports.updateArticle = function(article){
        return $http({
                method: 'put',
                url: '/article/put/' + article.title,
                data: article
            })
    };

    exports.registerUser = function(email, name, password){
        return $http({
            method: 'POST',
            url: '/user/register',
            data: {
                email: email,
                name: name,
                password: password
            }
        })
    };

    exports.loginUser = function(email, password){
        return $http({
            method: 'POST',
            url: '/user/login',
            data: {
                email: email,
                password: password
            }
        })
    };

    exports.logoutUser = function(){
        return $http({
            method: 'POST',
            url: '/user/logout'
        })
    };

    exports.getUser = function(){
        return $http({
            method: 'GET',
            url: '/user/get'
        })
    };

    exports.getAllArticles = function(){
        return $http({
            method: 'POST',
            url: '/articles/all'
        })
    };

    exports.getArticlePreview = function(article){
        return $http({
            method: 'POST',
            url: '/articles/preview',
            data: article
        })
    };

    exports.getArticleHistory = function(article){
        return $http({
            method: 'GET',
            url: '/history/get/'+article
        })
    };

    exports.getArchivedArticle = function(title, time){
        return $http({
            method: 'GET',
            url: '/archives/get/'+title+"/"+time,
        })
    };

    return exports;
}]);
