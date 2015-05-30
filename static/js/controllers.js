app.controller("ArticleViewCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, ArticleFactory){
        title = $routeParams.title;
        $scope.article = {}

        ArticleFactory.getArticle('html', title).
            success(function(data, status, headers, config) {
                $scope.body = data;
                $scope.article = {
                    title: title,
                    body: data
                }
            }).
            error(function(data, status, headers, config) {
                console.log("redirecting")
                $location.path('/' + title + '/edit')
            });

        $scope.getHtmlBody = function(){
            return $sce.trustAsHtml($scope.article.body);
        }

        $scope.registerUser = function(){
            ArticleFactory.registerUser().
                success(function(data){
                    console.log("Register succeeded")
                }).
                error(function(data){
                    console.log("Register failed")
                });
        }

}]);

app.controller('ArticleEditCtrl', ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$window',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $window, ArticleFactory){
        title = $routeParams.title;
        $scope.article = {}

        ArticleFactory.getArticle('markdown', title).
            success(function(data, status, headers, config) {
                $scope.article = {
                    title: title,
                    summary: "",
                    body: data
                }
            }).
            error(function(data, status, headers, config) {
                $scope.error = "Could not retrieve article"
                $scope.article = {
                    title: title,
                    summary: "",
                    body: ""
                }
            });

        $scope.update = function(article){
            ArticleFactory.updateArticle(article).
                success(function(data, status, headers, config) {
                    $scope.viewArticle()
                }).
                error(function(data, status, headers, config) {
                    console.log("Couldn't update article");
                });
        };

        $scope.viewArticle = function() {
            $location.path('/'+title);
        };
    }]);
