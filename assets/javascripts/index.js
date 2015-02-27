$(document).ready(function() {

  function sideNav() {
    $('#nav-toggle').sidr({
      name: 'main-nav',
      source: '#navigation',
      side: 'right'
    });

    $('#sidr-id-nav-close').click(function () {
      $.sidr('close', 'main-nav');
    });

    $(window).touchwipe({
      wipeRight: function () {
        $.sidr('close', 'main-nav');
      },
      wipeLeft: function () {
        $.sidr('open', 'main-nav');
      },
      preventDefaultEvents: false
    });
  }

  function heroScroll(){
    var $header = $('header');
    var $hero = $('.hero');
    var height = $hero.css("height");


    var desktopSize = window.matchMedia('all and (min-width: 650px)');
    var mobileSize = window.matchMedia('all and (max-width: 649px)');

    if (desktopSize.matches) {
      $('section:nth-of-type(2)').css('margin-top', height);

      $(window).scroll(function () {
        if ($(this).scrollTop() > (height.replace('px', '') - 80)) {
          $header.addClass('small');
        } else {
          $header.removeClass('small');
        }
      });
    }
    $(window).resize(function() {
      if (mobileSize.matches) {
        $('section:nth-of-type(2)').css('margin-top', '0');
      } else {
        $('section:nth-of-type(2)').css('margin-top', height);
      }
    });
  }

  sideNav();
  heroScroll();

});