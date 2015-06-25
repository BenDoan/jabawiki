app.controller("ArticleViewCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
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
                if (status === 401) {
                    $scope.error = "Not allowed, please login"
                }else{
                    $location.path('/w/' + title + '/edit');
                }
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

        $scope.loginUser = function(){
            ArticleFactory.loginUser().
                success(function(data){
                    console.log("Login succeeded")
                }).
                error(function(data){
                    console.log("Login failed")
                });

            $timeout(function(){
                $location.path('/');
            }, 500);
        }

        $scope.logoutUser = function(){
            ArticleFactory.logoutUser().
                success(function(data){
                    console.log("Logout succeeded")
                }).
                error(function(data){
                    console.log("Logout failed")
                });
            $timeout(function(){

                $location.path('/');
            }, 500);
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
            $location.path('/w/'+title);
        };
    }]);

app.controller("LoginCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
        $scope.login = function(article){
            ArticleFactory.loginUser($scope.email, $scope.password).
                success(function(data, status, headers, config) {
                    $timeout(function(){
                        $location.path('/');
                    }, 500);
                }).
                error(function(data, status, headers, config) {
                    $scope.error = data;
                });
        };

        $scope.register = function(article){
            if ($scope.reg_password != $scope.reg_password2){
                $scope.error = "Passwords do not match";
                return
            }

            ArticleFactory.registerUser($scope.reg_email, $scope.reg_name, $scope.reg_password).
                success(function(data, status, headers, config) {
                    $scope.error = "Success! Please log in";
                    $scope.reg_email = '';
                    $scope.reg_name = '';
                    $scope.reg_password = '';
                    $scope.reg_password2 = '';
                }).
                error(function(data, status, headers, config) {
                    $scope.error = data;
                });
        };
    }]);
