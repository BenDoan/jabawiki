var app = angular.module('wiki', ['ngRoute', 'angular-loading-bar']).
    config(['$routeProvider', '$locationProvider',
    function($routeProvider, $locationProvider){
        $routeProvider.
            when('/', {
                redirectTo: '/Home'
            }).
            when('/:title/edit', {
                templateUrl: 'partials/edit.html',
                controller: 'ArticleEditCtrl'
            }).
            when('/:title', {
                templateUrl: 'partials/view.html',
                controller: 'ArticleViewCtrl'
            }).
            otherwise({
                redirectTo: '/'
            });

        $locationProvider.html5Mode(true);
}]);

