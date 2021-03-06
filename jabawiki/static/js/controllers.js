app.controller("MasterCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){

        $scope.$on('$locationChangeSuccess',
            function(event, toState, toParams, fromState, fromParams){
                $scope.error = false
            })

        ArticleFactory.getUser().
            success(function(data, status, headers, config){
                $scope.user = data
            }).
            error(function(data, status, headers, config){
                console.log("Error fetching user")
            });

        $scope.logoutUser = function(){
            ArticleFactory.logoutUser().
                success(function(data){
                    console.log("Logout succeeded")
                    $scope.error = ["Logout Successful", "success"];
                    $scope.user = null
                }).
                error(function(data){
                    console.log("Logout failed")
                });
            $timeout(function(){
                $location.path('/');
            }, 500);
        }
    }]);

app.controller("ArticleViewCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
        $scope.$parent.page = "view"
        title = $routeParams.title;
        $scope.$parent.title = title
        $scope.display_title = title.replace(/_/g, " ");

        $scope.article = {};

        ArticleFactory.getArticle(title).
            success(function(data, status, headers, config) {
                $scope.article = {
                    title: title,
                    body: data.Body,
                    permission: data.Permission
                }
                $scope.$parent.article = $scope.article

            }).
            error(function(data, status, headers, config) {
                if (status === 401) {
                    $scope.$parent.error = ["Not allowed, please login", "warning"]
                }else{
                    $location.path('/w/' + title + '/edit');
                }
            });

        function format_wikilinks(txt){
            replacement_str = "[$1](http://" + window.location.host + "/w/$1)"
            return txt.replace(/\[\[([a-zA-Z0-9_]+)\]\]/g, replacement_str)
        }

        $scope.getHtmlBody = function(){
            if ($scope.article.body != null){
                return $sce.trustAsHtml(marked(format_wikilinks($scope.article.body)));
            }else{
                console.log("Null article body");
                return null;
            }
        }


}]);

app.controller('ArticleEditCtrl', ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$window',
                                   '$sce',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $window, $sce, ArticleFactory){
        $scope.$parent.page = "edit"
        title = $routeParams.title;
        $scope.$parent.title = title
        $scope.article = {}

        ArticleFactory.getArticle(title).
            success(function(data, status, headers, config) {
                $scope.article = {
                    title: title,
                    summary: "",
                    permission: data.Permission,
                    body: data.Body
                }
                $scope.$parent.article = $scope.article
            }).
            error(function(data, status, headers, config) {
                if (status === 401) {
                    $scope.$parent.error = ["Not allowed, please login", "warning"]
                }else if (status == 404) {
                    $scope.$parent.error = ["Article does not exist, try making it ", "danger"]
                    $scope.article = {
                        title: title,
                        summary: "",
                        permission: "private",
                        body: ""
                    }
                }

            });

        $scope.update = function(article, redirect){
            ArticleFactory.updateArticle(article).
                success(function(data, status, headers, config) {
                    $scope.$parent.error = ["Article saved", "success"]
                    if (redirect){
                        $scope.viewArticle()
                    }else{
                        $scope.article.summary = ""
                    }
                }).
                error(function(data, status, headers, config) {
                    console.log("Couldn't update article");
                    $scope.$parent.error = ["Error while updating article", "danger"]
                });
        };

        $scope.viewArticle = function() {
            $location.path('/w/'+title);
        };

        $scope.getPreview = function(article){
            ArticleFactory.getArticlePreview(article).
                success(function(data, status, headers, config){
                    console.log("data is: " + data.Body)
                    $scope.preview = $sce.trustAsHtml(data.Body);
                }).
                error(function(data, status, headers, config){
                    console.log("Couldn't get article preview");
                });
        }
    }]);

app.controller("LoginCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
        $scope.$parent.page = "login"
        $scope.login = function(article){
            ArticleFactory.loginUser($scope.email, $scope.password).
                success(function(data, status, headers, config) {
                    ArticleFactory.getUser().
                        success(function(data, status, headers, config){
                            $scope.$parent.user = data
                        }).
                        error(function(data, status, headers, config){
                            console.log("Error fetching user")
                        });

                    $timeout(function(){
                        $location.path('/');
                    }, 500);
                }).
                error(function(data, status, headers, config) {
                    $scope.$parent.error = [data, "danger"];
                });

        };

        $scope.register = function(article){
            if ($scope.reg_password != $scope.reg_password2){
                $scope.$parent.error = ["Passwords do not match", "danger"];
                return
            }

            ArticleFactory.registerUser($scope.reg_email, $scope.reg_name, $scope.reg_password).
                success(function(data, status, headers, config) {
                    $scope.$parent.error = ["Success! Please log in", "success"];
                    $scope.reg_email = '';
                    $scope.reg_name = '';
                    $scope.reg_password = '';
                    $scope.reg_password2 = '';
                }).
                error(function(data, status, headers, config) {
                    $scope.$parent.error = [data, "danger"];
                });
        };
    }]);

app.controller("ProfileCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
        $scope.$parent.page = "profile"
    }]);

app.controller("IndexCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
            $scope.$parent.page = "index"
            ArticleFactory.getAllArticles().
                success(function(data, status, headers, config) {
                    $scope.articles = data
                }).
                error(function(data, status, headers, config) {
                    $scope.$parent.error = ["Couldn't get article listing", "danger"];
                });

    }]);

app.controller("HistoryCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
            $scope.$parent.page = 'history'
            $scope.title = $routeParams.title;
            $scope.$parent.title = $scope.title;

            ArticleFactory.getArticleHistory($scope.title).
                success(function(data, status, headers, config) {
                    $scope.history = data
                }).
                error(function(data, status, headers, config) {
                    console.log("Error getting article history: " + data)
                });

            $scope.view = function(title, histItem){
                ArticleFactory.getArchivedArticle(title, histItem.time).
                    success(function(data, status, headers, config){
                        $scope.preview = $sce.trustAsHtml(data);
                        $scope.previewHistItem = histItem
                    }).
                    error(function(data, status, headers, config){
                        console.log("Couldn't get archived article: " + data);
                    });
            }

            $scope.revert = function(title, histItem){
                ArticleFactory.getArchivedArticle(title, histItem.time, "markdown").
                    success(function(data, status, headers, config){
                        message = "Reverted to \"" + histItem.summary+"\""
                        revert_data = {"title": title, "body": data, "summary": message}
                        ArticleFactory.updateArticle(revert_data).
                            success(function(data, status, headers, config) {
                                $scope.$parent.error = [message, "success"]
                            }).
                            error(function(data, status, headers, config) {
                                console.log("Couldn't update article");
                                $scope.$parent.error = ["Error while updating article: "+data, "danger"]
                            });

                    }).
                    error(function(data, status, headers, config){
                        console.log("Couldn't get archived article: " + data);
                    });
            }
    }]);

app.controller("UploadImageCtrl", ['$scope',
                                   '$routeParams',
                                   '$location',
                                   '$sce',
                                   '$timeout',
                                   'ArticleFactory',
    function($scope, $routeParams, $location, $sce, $timeout, ArticleFactory){
        $scope.$parent.page = 'uploadimage'
    }]);
