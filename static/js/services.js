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

    return exports;
}]);
